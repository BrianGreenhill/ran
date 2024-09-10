package activity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

type Service struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewService(db *sql.DB, logger *slog.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

func (a *Service) Add(ctx context.Context, activity Activity) error {
	sha := sha256.Sum256(activity.GPX)
	hash := hex.EncodeToString(sha[:])

	existingRow := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM activities WHERE gpx_hash = ?", hash)
	var count int
	if err := existingRow.Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		_, err := a.db.ExecContext(ctx, "DELETE FROM activities WHERE gpx_hash = ?", hash)
		if err != nil {
			return err
		}
		a.logger.Info("Deleted existing activity", slog.String("hash", hash))
	}

	g, err := gpx.ParseBytes(activity.GPX)
	if err != nil {
		return err
	}

	activitySplits := calculateSplits(*g)
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(activitySplits); err != nil {
		return err
	}

	res, err := a.db.ExecContext(ctx, `
    INSERT INTO activities
    (name,
    date,
    distance,
    duration,
    elevation_gain,
    average_pace,
    splits,
    elevation,
    gpx,
    gpx_hash)
    VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		activity.Name,
		activity.CompletedDate,
		activity.Distance,
		activity.Time,
		activity.Uphill,
		activity.AveragePace,
		buffer.Bytes(),
		activity.Elevation,
		activity.GPX,
		hash,
	)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		return fmt.Errorf("expected 1 row to be affected, got %d", affected)
	}

	return nil
}

func (a *Service) Get(ctx context.Context) ([]Activity, error) {
	rows, err := a.db.QueryContext(ctx,
		"SELECT name, date, distance, duration, elevation_gain, average_pace, splits, elevation, gpx, created_at FROM activities")

	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, err
	}

	var completedDateVal string
	var splitsVal []byte
	activities := []Activity{}
	for rows.Next() {
		activity := Activity{}
		if err := rows.Scan(&activity.Name, &completedDateVal, &activity.Distance, &activity.Time, &activity.Uphill, &activity.AveragePace, &splitsVal, &activity.Elevation, &activity.GPX, &activity.Created); err != nil {
			return nil, err
		}

		completedDate, err := time.Parse("2006-01-02 15:04:05+00:00", completedDateVal)
		if err != nil {
			return nil, err
		}

		activity.CompletedDate = &completedDate
		var splits []Split
		buffer := bytes.NewBuffer(splitsVal)
		dec := gob.NewDecoder(buffer)
		if err := dec.Decode(&splits); err != nil {
			return nil, err
		}

		activity.Splits = splits
		activities = append(activities, activity)
	}

	return activities, nil
}

func calculateSplits(g gpx.GPX) []Split {
	var splits []Split
	totalDistance := 0.0 // Total distance covered in meters
	var startTime *time.Time

	for _, track := range g.Tracks {
		for _, segment := range track.Segments {
			for i := 1; i < len(segment.Points); i++ {
				s := Split{}
				point1 := segment.Points[i-1]
				point2 := segment.Points[i]

				// Calculate distance between two points (in meters)
				distance := point1.Distance2D(&point2)
				totalDistance += distance

				// Set start time if it's the first point
				if startTime == nil {
					startTime = &point1.Timestamp
				}

				// Check if we have crossed the 1 km threshold
				for totalDistance >= 1000 {
					// Calculate the time difference for the last full kilometer
					endTime := point2.Timestamp
					timeDiff := endTime.Sub(*startTime)
					s.Distance = 1000
					s.SplitTime = timeDiff.Seconds()

					s.Elevation = point2.Elevation.Value()

					// Append split time (in seconds)
					splits = append(splits, s)

					// Reset start time for the next kilometer
					startTime = &point2.Timestamp

					// Subtract 1 km (1000 meters) from totalDistance, leave the remaining distance
					totalDistance -= 1000
				}
			}
		}
	}

	// Handle any remaining partial distance (if the run is not an exact number of kilometers)
	if totalDistance > 0 {
		endTime := g.Tracks[0].Segments[0].Points[len(g.Tracks[0].Segments[0].Points)-1].Timestamp
		timeDiff := endTime.Sub(*startTime)
		s := Split{}
		s.Distance = totalDistance
		s.SplitTime = timeDiff.Seconds()
		s.Elevation = g.Tracks[0].Segments[0].Points[len(g.Tracks[0].Segments[0].Points)-1].Elevation.Value()
		splits = append(splits, s)
	}

	return splits
}
