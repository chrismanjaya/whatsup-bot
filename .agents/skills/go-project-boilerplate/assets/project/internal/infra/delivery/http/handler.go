// Package http is the HTTP delivery adapter: it decodes requests into usecase
// inputs, calls the usecase, and maps results and error codes to responses.
// It is the boundary where errors are logged, exactly once.
package http

import (
	"context"
	"encoding/json"
	"net/http"

	"example.com/app/internal/constant"
	"example.com/app/internal/domain"
	"example.com/app/internal/usecase/example"
	"example.com/app/internal/util"
)

const maxBodyBytes = 1 << 20

// CreateExampleUsecase is the port this handler needs; *example.Create
// satisfies it, and tests can pass a fake.
type CreateExampleUsecase interface {
	Execute(ctx context.Context, input example.CreateInput) (*domain.Example, error)
}

type Handler struct {
	createExample CreateExampleUsecase
}

func NewHandler(createExample CreateExampleUsecase) *Handler {
	return &Handler{createExample: createExample}
}

// Routes registers every endpoint and wraps them in the standard middleware.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /examples", h.postExample)
	return withRequestID(withRecover(mux))
}

type (
	createExampleRequest struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	exampleResponse struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	errorResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) postExample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// decode request
	var req createExampleRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(ctx, w, util.WrapStd(constant.ErrInvalidRequest, "decode request failed", err))
		return
	}

	// execute usecase
	created, err := h.createExample.Execute(ctx, example.CreateInput{
		Name:   req.Name,
		Status: req.Status,
	})
	if err != nil {
		writeError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, http.StatusCreated, exampleResponse{
		ID:     created.ID,
		Name:   created.Name,
		Status: created.Status.String(),
	})
}

// writeError logs the full error chain once and answers with only the code
// and its generic message, so internal detail never leaks to clients.
func writeError(ctx context.Context, w http.ResponseWriter, err error) {
	code := util.CodeOrInternal(err)
	status := httpStatus(code)

	if status >= http.StatusInternalServerError {
		util.LogError(ctx, "request failed", err)
	} else {
		util.LogWarn(ctx, "request rejected", err)
	}

	writeJSON(ctx, w, status, errorResponse{Code: code.Code, Message: code.Message})
}

// httpStatus maps an error code to an HTTP status: common codes are already
// HTTP statuses; domain codes (DDNNN) are client errors by default.
func httpStatus(code *constant.ErrCode) int {
	if code.Code >= 400 && code.Code < 600 {
		return code.Code
	}
	return http.StatusUnprocessableEntity
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		util.LogWarn(ctx, "write response failed", err)
	}
}
