package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
)

// maxRequestBodyBytes caps every JSON request body. The largest
// legitimate payload (a listing or forum post with its description) is
// a few KB, and photos never transit the API (they go straight to
// object storage through presigned URLs), so 1 MB is generous while
// still stopping a client from streaming an unbounded body into memory.
const maxRequestBodyBytes = 1 << 20

// readJSON decodes r's body into dst, writing the error response itself
// when that fails: 413 for an oversized body, 400 for anything that
// isn't valid JSON. Callers just `return` when it reports false.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "request body too large"})
			return false
		}
		writeError(w, errBadRequestBody)
		return false
	}
	return true
}
