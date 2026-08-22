package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tank-ocr/learn"
)

func TestHandleLearnInterestFootball(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/learn/interest?q=%E3%82%B5%E3%83%83%E3%82%AB%E3%83%BC", nil)
	rec := httptest.NewRecorder()
	HandleLearnInterest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var result learn.MatchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Stories) == 0 {
		t.Fatal("expected stories")
	}
}

func TestHandleLearnCatalog(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/learn/catalog", nil)
	rec := httptest.NewRecorder()
	HandleLearnCatalog(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var cat learn.Catalog
	if err := json.Unmarshal(rec.Body.Bytes(), &cat); err != nil {
		t.Fatal(err)
	}
	if len(cat.Stories) < 3 {
		t.Fatalf("stories = %d", len(cat.Stories))
	}
}

func TestHandleLearnInterestPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/learn/interest", strings.NewReader(`{"text":"C罗"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	HandleLearnInterest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var result learn.MatchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Stories) == 0 || result.Stories[0].ID != "ronaldo-childhood" {
		t.Fatalf("got %#v", result.Stories)
	}
}

func TestHandleLearnStoryNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/learn/story/missing", nil)
	rec := httptest.NewRecorder()
	HandleLearnStory(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}
