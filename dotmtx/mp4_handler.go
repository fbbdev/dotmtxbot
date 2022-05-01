package dotmtx

import "net/http"

func Mp4Handler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}
