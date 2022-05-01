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
	speed, err := strconv.ParseFloat(r.URL.Query().Get("speed"), 64)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	width, err := strconv.ParseFloat(r.URL.Query().Get("width"), 64)
	if err != nil || width <= 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	blank, err := strconv.ParseFloat(r.URL.Query().Get("blank"), 64)
	if err != nil || blank < 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(r.URL.Query().Get("text")) > MaxChars {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// log.InfoLogger.Print("parameters are valid")

	anim, err := MakeGif(speed, width, blank, r.URL.Query().Get("text"))
	if err != nil {
		log.ErrorLogger.Print("MakeGif: ", err)
		log.WarningLogger.Print("gif generation failed")
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "image/gif")
	w.Header().Add("Cache-Control", "max-age=1, s-maxage=3600, public, immutable, stale-while-revalidate")
	err = gif.EncodeAll(w, anim)
	if err != nil {
		log.ErrorLogger.Print("gif/http: ", err)
		log.WarningLogger.Print("could not encode gif or write http response")
		// just in case EncodeAll did not write anything
		w.WriteHeader(http.StatusInternalServerError)
	}
}
