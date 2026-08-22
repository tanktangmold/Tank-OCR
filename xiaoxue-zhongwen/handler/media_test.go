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
