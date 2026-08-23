package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"xiaoxue-zhongwen/learn"
)

func TestHandleVoicesHasMaleAndFemale(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/voices", nil)
	rec := httptest.NewRecorder()
	HandleVoices(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var voices []learn.Voice
	if err := json.Unmarshal(rec.Body.Bytes(), &voices); err != nil {
		t.Fatal(err)
	}
	var female, male int
	for _, voice := range voices {
		if voice.Gender == "female" {
			female++
		}
		if voice.Gender == "male" {
			male++
		}
	}
	if female < 2 || male < 2 {
		t.Fatalf("female=%d male=%d", female, male)
	}
}

func TestHandleTTSRequiresText(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tts?voice=xiaoxiao", nil)
	rec := httptest.NewRecorder()
	HandleTTS(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandleSentenceImageServesLocalSVG(t *testing.T) {
	learn.SetMediaPaths(t.TempDir(), "static")
	req := httptest.NewRequest(http.MethodGet, "/api/image?day=1&line=0&style=comic", nil)
	rec := httptest.NewRecorder()
	HandleSentenceImage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Fatalf("content type %q", got)
	}
}
