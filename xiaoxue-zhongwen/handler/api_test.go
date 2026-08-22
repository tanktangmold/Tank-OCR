package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"xiaoxue-zhongwen/learn"
)

func TestHandleInterestFootball(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/interest?q=%E3%82%B5%E3%83%83%E3%82%AB%E3%83%BC", nil)
	rec := httptest.NewRecorder()
	HandleInterest(rec, req)
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

func TestHandleCatalog(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	rec := httptest.NewRecorder()
	HandleCatalog(rec, req)
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

func TestHandleInterestPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/interest", strings.NewReader(`{"text":"C罗"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	HandleInterest(rec, req)
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

func TestHandleStoryNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/story/missing", nil)
	rec := httptest.NewRecorder()
	HandleStory(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}
