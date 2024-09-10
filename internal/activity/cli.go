package activity

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

type CLI struct {
	writer          io.Writer
	db              *sql.DB
	activityService *Service
	args            []string
	logger          *slog.Logger
}

func NewCLI(w io.Writer, db *sql.DB, logger *slog.Logger, activityService *Service, args []string) *CLI {
	return &CLI{
		writer:          w,
		db:              db,
		activityService: activityService,
		args:            args,
		logger:          logger,
	}
}

func (c *CLI) Run(args []string) error {
	if len(args) == 0 {
		c.Usage()
		return nil
	}

	switch args[0] {
	case "add":
		if err := c.AddActivity(); err != nil {
			return err
		}
	case "api":
		if err := c.RunAPI(context.Background()); err != nil {
			return err
		}
	default:
		c.Usage()
	}
	return nil
}

func (c *CLI) Usage() {
	fmt.Fprintf(c.writer, "Usage: ran [command] [flags]\n--help show this message\n\n\tadd --gpx\n\tview\n")
}

func (c *CLI) RunAPI(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	mux := NewAPI(c.logger, c.activityService)

	server := &http.Server{
		Addr:    ":8222",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		c.logger.Info("Shutting down server")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			c.logger.Error("Error shutting down server", slog.Any("error", err))
		}
	}()

	c.logger.Info("Starting server")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		c.logger.Error("Error starting server", slog.Any("error", err))
		cancel()
		return err
	}

	return nil
}

func (c *CLI) AddActivity() error {
	fs := flag.NewFlagSet("ran", flag.ExitOnError)
	var gpxFile string
	fs.StringVar(&gpxFile, "gpx", "", "path to gpx file")
	fs.Usage = c.Usage

	if err := fs.Parse(c.args[1:]); err != nil {
		return err
	}

	if gpxFile == "" {
		fs.Usage()
	}

	c.logger.Info("gpx file: %s", slog.String("gpx_file", gpxFile))

	gpxBytes, err := readGPXFile(gpxFile)
	if err != nil {
		return err
	}

	g, err := gpx.ParseBytes(gpxBytes)
	if err != nil {
		return err
	}

	distanceKm := g.MovingData().MovingDistance / 1000.0
	averageMinPerKm := (g.MovingData().MovingTime / distanceKm) / 60

	startingElevation := g.Tracks[0].Segments[0].Points[0].Elevation.Value()

	activity := Activity{
		Name:          g.Name,
		GPX:           gpxBytes,
		Distance:      g.MovingData().MovingDistance,
		Time:          g.Duration(),
		CompletedDate: g.Time,
		Elevation:     startingElevation,
		Uphill:        g.UphillDownhill().Uphill,
		Downhill:      g.UphillDownhill().Downhill,
		AveragePace:   averageMinPerKm,
	}

	if g.Name == "" && g.Description == "" {
		if g.Time != nil {
			if g.Time.Hour() >= 12 && g.Time.Hour() <= 18 {
				activity.Name = "Afternoon Run"
			} else if g.Time.Hour() >= 18 {
				activity.Name = "Night Run"
			} else {
				activity.Name = "Morning Run"
			}
		}
	}

	if err := c.activityService.Add(context.Background(), activity); err != nil {
		return err
	}

	fmt.Fprintln(c.writer, "Activity added successfully")

	return nil
}

func readGPXFile(gpxFile string) ([]byte, error) {
	info, err := os.Stat(gpxFile)
	if err != nil {
		return nil, fmt.Errorf("error reading gpx file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("gpx file is a directory")
	}

	file, err := os.Open(gpxFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return contents, nil
}
