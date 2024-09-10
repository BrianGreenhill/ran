package main

import (
	"database/sql"
	"io"
	"log/slog"
	"os"

	"github.com/briangreenhill/ran/internal/activity"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	w := os.Stdout
	logger := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{}))
	if _, err := os.Stat("ran.db"); err != nil {
		err := os.WriteFile("ran.db", []byte(""), 0644)
		if err != nil {
			logger.Error("Error creating database file", slog.Any("file", "ran.db"))
			os.Exit(1)
		}
	}

	db, err := sql.Open("sqlite3", "./ran.db")
	if err != nil {
		logger.Error("Error opening database", slog.Any("error", err))
		os.Exit(1)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS activities (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT,
        date TEXT,
        distance REAL,
        duration REAL,
        elevation_gain REAL,
        average_pace REAL,
        elevation REAL,
        gpx BLOB,
        gpx_hash TEXT UNIQUE,
        splits BLOB,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil {
		logger.Error("Error creating table", slog.Any("error", err))
		os.Exit(1)
	}

	activityService := activity.NewService(db, logger)

	if err := run(w, os.Args[1:], db, logger, activityService); err != nil {
		logger.Error("Error running ran", slog.Any("error", err))
		return
	}
}

func run(w io.Writer, args []string, db *sql.DB, logger *slog.Logger, activityService *activity.Service) error {
	cli := activity.NewCLI(w, db, logger, activityService, args)

	if err := cli.Run(args); err != nil {
		return err
	}

	return nil
}
