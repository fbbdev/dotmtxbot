package dotmtx

import (
	"image/gif"
	"net/http"
	"os"
	"os/exec"
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

	ffmpeg := exec.CommandContext(
		r.Context(),
		"ffmpeg",
		// "-hide_banner",
		// "-loglevel", "error",
		"-f", "gif",
		"-i", "pipe:",
		"-movflags", "frag_keyframe+empty_moov",
		"-pix_fmt", "yuv420p",
		"-f", "mp4",
		"pipe:",
	)

	ffmpegInput, err := ffmpeg.StdinPipe()
	if err != nil {
		log.ErrorLogger.Print("exec: ", err)
		log.WarningLogger.Print("ffmpeg input pipe creation failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ffmpeg.Stdout = w
	ffmpeg.Stderr = os.Stderr

	err = ffmpeg.Start()
	if err != nil {
		log.ErrorLogger.Print("exec: ", err)
		log.WarningLogger.Print("ffmpeg startup failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	go func() {
		defer ffmpegInput.Close()
		if err := gif.EncodeAll(ffmpegInput, anim); err != nil {
			log.ErrorLogger.Print("gif: ", err)
			log.WarningLogger.Print("gif encoding and transfer to ffmpeg failed")
			ffmpeg.Process.Kill()
		}
	}()

	w.Header().Add("Content-Type", "video/mp4")
	err = ffmpeg.Wait()
	if err != nil {
		log.ErrorLogger.Print("exec: ", err)
		log.WarningLogger.Print("gif to mp4 conversion failed")
		// just in case nothing was written
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
