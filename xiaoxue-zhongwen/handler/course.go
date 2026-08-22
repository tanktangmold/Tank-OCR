package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"os"

	"xiaoxue-zhongwen/learn"
)

func HandleCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, learn.LoadCourse())
}

func HandleCourseDay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("day")
	var day int
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			writeJSONError(w, http.StatusBadRequest, "invalid day")
			return
		}
		day = day*10 + int(ch-'0')
	}
	lesson, ok := learn.DayByNumber(day)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "day not found")
		return
	}
	writeJSON(w, lesson)
}

func HandleChildren(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		children, err := learn.ListChildren()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, children)
	case http.MethodPost:
		var body struct {
			Name   string `json:"name"`
			Avatar string `json:"avatar"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		child, err := learn.AddChild(body.Name, body.Avatar)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, child)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleChildProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing child id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		progress, err := learn.GetProgress(id)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, progress)
	case http.MethodPost:
		var patch learn.ProgressPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		progress, err := learn.ApplyProgress(id, patch)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, progress)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	childID := r.URL.Query().Get("child")
	dayStr := r.PathValue("day")
	day := 0
	for _, ch := range dayStr {
		if ch < '0' || ch > '9' {
			writeJSONError(w, http.StatusBadRequest, "invalid day")
			return
		}
		day = day*10 + int(ch-'0')
	}
	if childID != "" {
		progress, err := learn.GetProgress(childID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		state := progress.Days[dayStr]
		if state == nil || !state.LessonDone {
			writeJSONError(w, http.StatusForbidden, "finish the 15-minute lesson first")
			return
		}
	}
	game, err := learn.BuildGame(day)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, game)
}

func HandleServerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	port := "8080"
	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		if _, p, err := net.SplitHostPort(addr); err == nil && p != "" {
			port = p
		}
	}
	host := r.Host
	if host == "" {
		host = "127.0.0.1:" + port
	}
	writeJSON(w, map[string]any{
		"name":    "小小中文",
		"port":    port,
		"thisUrl": "http://" + host,
		"lanUrls": learn.LANURLs(port),
		"hintJa":  "同じWi-FiのiPadで、このアドレスを開いてね。子どもは自分の顔を押せばいいよ。",
		"hintZh":  "同一Wi-Fi下的iPad打开这个地址。每个孩子点自己的头像，进度互不影响。",
	})
}
