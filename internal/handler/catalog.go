package handler

import (
	"net/http"
	"path"

	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

type pieceBounds struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type puzzlePieceResponse struct {
	ID       string      `json:"id"`
	ImageKey string      `json:"image_key"`
	SVGPath  string      `json:"svg_path"`
	GridX    int         `json:"grid_x"`
	GridY    int         `json:"grid_y"`
	Bounds   pieceBounds `json:"bounds"`
}

type gameCatalogResponse struct {
	catalogPuzzleResponse
	Pieces []puzzlePieceResponse `json:"pieces"`
	Reward *rewardResponse       `json:"reward,omitempty"`
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// HandleListCatalog GET /api/catalog
func (h *Handler) HandleListCatalog(w http.ResponseWriter, r *http.Request) {
	locale := middleware.LocaleFromContext(r.Context())
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty != "" && difficulty != "easy" && difficulty != "medium" && difficulty != "hard" {
		writeError(w, http.StatusBadRequest, "invalid difficulty value")
		return
	}
	filters := store.CatalogFilters{
		CategorySlug: r.URL.Query().Get("category"),
		Difficulty:   difficulty,
	}
	list, err := h.Store.ListPublicCatalog(r.Context(), locale, filters)
	if err != nil {
		h.Log.Error("list catalog", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]catalogPuzzleResponse, 0, len(list))
	for _, cp := range list {
		resp = append(resp, catalogPuzzleToResponse(cp))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetCatalogPuzzle GET /api/catalog/{id}
func (h *Handler) HandleGetCatalogPuzzle(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	cp, err := h.Store.GetCatalogPuzzle(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("get catalog puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := gameCatalogResponse{catalogPuzzleResponse: catalogPuzzleToResponse(cp)}

	if cp.Status == "ready" {
		pieces, err := h.Store.GetPuzzlePieces(r.Context(), cp.PuzzleID)
		if err != nil {
			h.Log.Error("get puzzle pieces", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		resp.Pieces = make([]puzzlePieceResponse, 0, len(pieces))
		for _, p := range pieces {
			resp.Pieces = append(resp.Pieces, puzzlePieceResponse{
				ID:       p.ID,
				ImageKey: p.ImageKey,
				SVGPath:  p.PathSVG,
				GridX:    p.GridX,
				GridY:    p.GridY,
				Bounds: pieceBounds{
					X: toInt(p.Bounds["x"]),
					Y: toInt(p.Bounds["y"]),
					W: toInt(p.Bounds["w"]),
					H: toInt(p.Bounds["h"]),
				},
			})
		}

		reward, err := h.Store.GetRewardByPuzzleID(r.Context(), cp.PuzzleID)
		if err == nil {
			rr := rewardResponse{
				ID:        reward.ID,
				PuzzleID:  reward.PuzzleID,
				VideoKey:  reward.VideoKey,
				Word:      reward.Word,
				TTSKey:    reward.TTSKey,
				Animation: reward.Animation,
			}
			resp.Reward = &rr
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
