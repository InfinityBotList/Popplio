package panel

import (
	_ "embed"
	"net/http"
)

// openapiDoc is a hand-written OpenAPI document.
//
// §4 offers two options; this is (a). The Rust service generated this from
// utoipa annotations, which Go has no equivalent of, and a codegen step would
// have to be maintained in lockstep with the union codec for no gain: the API is
// two routes and the request body is one union whose shape is already pinned by
// the round-trip tests.
//
//go:embed openapi.json
var openapiDoc []byte

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeText(http.StatusMethodNotAllowed, "Method Not Allowed").write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(openapiDoc)
}
