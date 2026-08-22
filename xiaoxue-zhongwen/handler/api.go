package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"xiaoxue-zhongwen/learn"
)

func HandleCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cat, err := learn.LoadCatalog()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, cat)
}

func HandleInterest(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if r.Method == http.MethodPost {
		var body struct {
			Query string `json:"query"`
			Q     string `json:"q"`
			Text  string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && r.Body != http.NoBody {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if query == "" {
			query = firstNonEmpty(body.Query, body.Q, body.Text)
		}
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := learn.MatchInterest(query)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, result)
}

func HandleBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, learn.SearchBooks(query))
}

func HandleStory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		id = strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/story/"), "/")
	}
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing story id")
		return
	}
	story, ok := learn.StoryByID(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "story not found")
		return
	}
	writeJSON(w, story)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": message})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
