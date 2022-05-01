package dotmtx

import (
	"image/gif"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"petbots.fbbdev.it/dotmtxbot/log"
)

func Mp4Handler(w http.ResponseWriter, r *http.Request) {
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

	tmpDir, err := ioutil.TempDir("", "dotmtxbot_mp4")
	if err != nil {
		log.ErrorLogger.Print("ioutil: ", err)
		log.WarningLogger.Print("temporary dir creation failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	gifPath := filepath.Join(tmpDir, "dotmtx.gif")
	mp4Path := filepath.Join(tmpDir, "dotmtx.mp4")

	gifFile, err := os.Create(gifPath)
	if err != nil {
		log.ErrorLogger.Print("os: ", err)
		log.WarningLogger.Print("temporary file creation failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = gif.EncodeAll(gifFile, anim)
	gifFile.Close()
	if err != nil {
		log.ErrorLogger.Print("gif: ", err)
		log.WarningLogger.Print("could not write GIF to temporary file")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ffmpeg := exec.CommandContext(
		r.Context(),
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", gifPath,
		"-movflags", "+faststart",
		"-pix_fmt", "yuv420p",
		mp4Path,
	)

	ffmpeg.Stderr = os.Stderr

	err = ffmpeg.Run()
	if err != nil {
		log.ErrorLogger.Print("exec: ", err)
		log.WarningLogger.Print("gif to mp4 conversion failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, mp4Path)
}
