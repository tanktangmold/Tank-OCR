package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"xiaoxue-zhongwen/learn"
)

func TestHandleCourseHasThirtyDays(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/course", nil)
	rec := httptest.NewRecorder()
	HandleCourse(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var course learn.Course
	if err := json.Unmarshal(rec.Body.Bytes(), &course); err != nil {
		t.Fatal(err)
	}
	if len(course.Days) != 30 {
		t.Fatalf("days = %d", len(course.Days))
	}
}

func TestHandleChildrenAndProgress(t *testing.T) {
	learn.SetClassroomPath(filepath.Join(t.TempDir(), "classroom.json"))
	req := httptest.NewRequest(http.MethodPost, "/api/children", strings.NewReader(`{"name":"ハル","avatar":"⚽"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	HandleChildren(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var child learn.Child
	if err := json.Unmarshal(rec.Body.Bytes(), &child); err != nil {
		t.Fatal(err)
	}

	progReq := httptest.NewRequest(http.MethodPost, "/api/children/"+child.ID+"/progress", strings.NewReader(`{"day":1,"lessonDone":true,"heard":[0,1]}`))
	progReq.Header.Set("Content-Type", "application/json")
	progReq.SetPathValue("id", child.ID)
	progRec := httptest.NewRecorder()
	HandleChildProgress(progRec, progReq)
	if progRec.Code != http.StatusOK {
		t.Fatalf("progress status %d %s", progRec.Code, progRec.Body.String())
	}

	gameReq := httptest.NewRequest(http.MethodGet, "/api/game/1?child="+child.ID, nil)
	gameReq.SetPathValue("day", "1")
	gameRec := httptest.NewRecorder()
	HandleGame(gameRec, gameReq)
	if gameRec.Code != http.StatusOK {
		t.Fatalf("game status %d %s", gameRec.Code, gameRec.Body.String())
	}
}
