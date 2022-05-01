package dotmtx

import (
	"image/gif"
	"net/http"
	"os"
	"strconv"

	"petbots.fbbdev.it/dotmtxbot/log"
)

var imgHost string

func init() {
	imgHost = os.Getenv("DOTMTXBOT_IMG_HOST")
	if imgHost == "" {
		imgHost = "localhost:3000"
	}
}

func GifHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	speed, err := strconv.ParseFloat(r.URL.Query().Get("speed"), 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	width, err := strconv.ParseFloat(r.URL.Query().Get("width"), 64)
	if err != nil || width <= 0 {
		http.NotFound(w, r)
		return
	}

	blank, err := strconv.ParseFloat(r.URL.Query().Get("blank"), 64)
	if err != nil || blank < 0 {
		http.NotFound(w, r)
		return
	}

	if len(r.URL.Query().Get("text")) > MaxChars {
		http.NotFound(w, r)
		return
	}

	// log.InfoLogger.Print("parameters are valid")

	anim, err := MakeGif(speed, width, blank, r.URL.Query().Get("text"))
	if err != nil {
		log.ErrorLogger.Print("MakeGif: ", err)
		log.WarningLogger.Print("gif generation failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "image/gif")
	err = gif.EncodeAll(w, anim)
	if err != nil {
		log.ErrorLogger.Print("gif/http: ", err)
		log.WarningLogger.Print("could not encode gif or write http response")
		// just in case nothing was written
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
