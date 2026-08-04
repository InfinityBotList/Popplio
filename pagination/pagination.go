// Package pagination parses the "page" query parameter shared by every
// paged list endpoint, so each handler doesn't hand-roll the same
// parse-with-default-of-1 logic (and, previously, a different error message
// per handler for the identical failure).
package pagination

import (
	"net/http"
	"strconv"
)

// Parse reads the "page" query parameter from r, defaulting to 1 when
// absent. It returns an error when the parameter is present but is not a
// valid non-negative integer.
func Parse(r *http.Request) (uint64, error) {
	page := r.URL.Query().Get("page")

	if page == "" {
		return 1, nil
	}

	return strconv.ParseUint(page, 10, 32)
}
