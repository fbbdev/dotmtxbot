package dotmtx

import "net/http"

func Mp4Handler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
