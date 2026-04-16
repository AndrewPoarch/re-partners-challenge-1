// Package api implements JSON HTTP handlers and wires them with Chi.
// Handlers delegate business rules to calculator and packsize; this layer only validates and maps errors to status codes.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/andreigliga/pack-calculator/internal/calculator"
	"github.com/andreigliga/pack-calculator/internal/packsize"
)

// MaxItemsPerRequest limits DP array size so a single request cannot exhaust memory.
const MaxItemsPerRequest = 100_000_000

// PackSizeService is the persistence-backed pack list; an interface so handlers stay testable with stubs.
type PackSizeService interface {
	Get(ctx context.Context) ([]int, error)
	Replace(ctx context.Context, sizes []int) ([]int, error)
}

type Handlers struct {
	Packs PackSizeService
}

type packSizesResponse struct {
	Sizes []int `json:"sizes"`
}

type packSizesRequest struct {
	Sizes []int `json:"sizes"`
}

type calculateRequest struct {
	Items int   `json:"items"`
	Sizes []int `json:"sizes,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handlers) GetPackSizes(w http.ResponseWriter, r *http.Request) {
	sizes, err := h.Packs.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sizes == nil {
		sizes = []int{}
	}
	writeJSON(w, http.StatusOK, packSizesResponse{Sizes: sizes})
}

func (h *Handlers) PutPackSizes(w http.ResponseWriter, r *http.Request) {
	var req packSizesRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	sizes, err := h.Packs.Replace(r.Context(), req.Sizes)
	if err != nil {
		if errors.Is(err, packsize.ErrInvalidSizes) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, packSizesResponse{Sizes: sizes})
}

func (h *Handlers) Calculate(w http.ResponseWriter, r *http.Request) {
	var req calculateRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.Items < 0 {
		writeError(w, http.StatusBadRequest, "items must be non-negative")
		return
	}
	if req.Items > MaxItemsPerRequest {
		writeError(w, http.StatusBadRequest, "items exceeds maximum allowed value")
		return
	}

	sizes := req.Sizes
	if len(sizes) == 0 {
		stored, err := h.Packs.Get(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sizes = stored
	}
	if len(sizes) == 0 {
		writeError(w, http.StatusBadRequest, "no pack sizes configured; set them via PUT /api/pack-sizes or pass `sizes` in the request body")
		return
	}

	result, err := calculator.Calculate(req.Items, sizes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func decodeJSON(body io.ReadCloser, into any) error {
	defer body.Close()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	return dec.Decode(into)
}
