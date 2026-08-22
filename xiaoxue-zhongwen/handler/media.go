package handler

import (
	"net/http"
	"strconv"
	"strings"

	"xiaoxue-zhongwen/learn"
)

func HandleVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, learn.Voices)
}

func HandleTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	text := strings.TrimSpace(r.URL.Query().Get("text"))
	voice := r.URL.Query().Get("voice")
	if text == "" {
		writeJSONError(w, http.StatusBadRequest, "missing text")
		return
	}
	if len([]rune(text)) > 80 {
		writeJSONError(w, http.StatusBadRequest, "text too long")
		return
	}
	path, err := learn.EnsureTTS(text, voice)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	serveMedia(w, r, path, "audio/mpeg")
}

func HandleSentenceImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	day, _ := strconv.Atoi(r.URL.Query().Get("day"))
	line, _ := strconv.Atoi(r.URL.Query().Get("line"))
	style := r.URL.Query().Get("style")
	lesson, ok := learn.DayByNumber(day)
	if !ok || line < 0 || line >= len(lesson.Lines) {
		writeJSONError(w, http.StatusNotFound, "line not found")
		return
	}
	beat := lesson.Lines[line]
	path, err := learn.EnsureSentenceImage(beat.Sentence.ZH, beat.Scene, style)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	ctype := "image/jpeg"
	if strings.HasSuffix(path, ".webp") {
		ctype = "image/webp"
	}
	serveMedia(w, r, path, ctype)
}

func serveMedia(w http.ResponseWriter, r *http.Request, path, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, path)
}
