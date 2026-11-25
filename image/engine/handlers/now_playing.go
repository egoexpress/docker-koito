package handlers

import (
	"net/http"
	"time"

	"github.com/gabehf/koito/internal/catalog"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/memkv"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/utils"
)

type NowPlayingResponse struct {
	CurrentlyPlaying bool         `json:"currently_playing"`
	Track            models.Track `json:"track"`
	// When the current play started. Omitted if unknown.
	StartedAt *time.Time `json:"started_at,omitempty"`
}

func NowPlayingHandler(store db.TrackStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("NowPlayingHandler: Got request")

		// Hardcoded user id as 1. Not great but it works until (if) multi-user is supported.
		if stateI, ok := memkv.Store.Get("1"); !ok {
			utils.WriteJSON(w, http.StatusOK, NowPlayingResponse{CurrentlyPlaying: false})
		} else if state, ok := stateI.(catalog.NowPlayingState); !ok {
			l.Debug().Msg("NowPlayingHandler: Failed type assertion for now-playing state")
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		} else {
			track, err := store.GetTrack(ctx, db.GetTrackOpts{ID: state.TrackID})
			if err != nil {
				l.Error().Err(err).Msg("NowPlayingHandler: Failed to get track from database")
				utils.WriteError(w, "failed to fetch currently playing track from database", http.StatusInternalServerError)
			} else {
				startedAt := state.StartedAt
				utils.WriteJSON(w, http.StatusOK, NowPlayingResponse{CurrentlyPlaying: true, Track: *track, StartedAt: &startedAt})
			}
		}
	}
}
