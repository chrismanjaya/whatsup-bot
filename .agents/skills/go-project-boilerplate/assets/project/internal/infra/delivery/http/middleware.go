package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"example.com/app/internal/constant"
	"example.com/app/internal/util"
)

const requestIDHeader = "X-Request-ID"

// withRequestID tags the request context's logger with a request_id, taken
// from the caller or generated, and echoes it back in the response.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" || len(id) > 64 {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)

		ctx := util.WithLogAttrs(r.Context(), "request_id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withRecover turns a panic into a logged 500 instead of a dropped connection.
func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				util.Logger(r.Context()).Error("handler panicked", "panic", p)
				writeJSON(r.Context(), w, http.StatusInternalServerError, errorResponse{
					Code:    constant.ErrInternal.Code,
					Message: constant.ErrInternal.Message,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
