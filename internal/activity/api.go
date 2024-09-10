package activity

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

func NewAPI(logger *slog.Logger, activityService *Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /gpx/{id}", handleGetActivities(logger, activityService))
	mux.Handle("GET /", http.FileServer(http.Dir("./ui")))
	mux.Handle("GET /gpx/{id}/detail", handleGetActivityDetail(logger, activityService))
	mux.Handle("GET /token", handleToken(logger))

	return mux
}

func handleToken(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := os.LookupEnv("MAPBOX_TOKEN")
		if !ok {
			logger.Error("Error getting token", slog.String("error", "MAPBOX_TOKEN not set"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		key := map[string]string{"token": token}
		if err := json.NewEncoder(w).Encode(key); err != nil {
			logger.Error("Error encoding token", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}

func handleGetActivityDetail(logger *slog.Logger, activityService *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		num, err := strconv.Atoi(id)
		if err != nil {
			logger.Error("Error converting id to int", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		activities, err := activityService.Get(r.Context())
		if err != nil {
			logger.Error("Error getting activities", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		activity := activities[num]

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(activity); err != nil {
			logger.Error("Error encoding activity", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}

func handleGetActivities(logger *slog.Logger, activityService *Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		num, err := strconv.Atoi(id)
		if err != nil {
			logger.Error("Error converting id to int", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		activities, err := activityService.Get(r.Context())
		if err != nil {
			logger.Error("Error getting activites", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		activity := activities[num]

		gpx := activity.GPX

		w.Header().Set("Content-Type", "application/gpx+xml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(gpx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}
