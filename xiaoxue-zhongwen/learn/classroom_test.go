package learn

import "testing"

func TestAddChildProgressIsIndependent(t *testing.T) {
	dir := t.TempDir()
	SetClassroomPath(dir + "/classroom.json")

	a, err := AddChild("ハル", "⚽")
	if err != nil {
		t.Fatal(err)
	}
	b, err := AddChild("ソラ", "🌟")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyProgress(a.ID, ProgressPatch{Day: 1, Heard: []int{0, 1, 2}, LessonDone: true, Seconds: 900}); err != nil {
		t.Fatal(err)
	}
	pa, err := GetProgress(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := GetProgress(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !pa.Days["1"].LessonDone {
		t.Fatal("child A should finish day 1")
	}
	if pb.Days["1"] != nil && pb.Days["1"].LessonDone {
		t.Fatal("child B should still be on day 1 locked progress")
	}
	if _, err := ApplyProgress(b.ID, ProgressPatch{Day: 2, LessonDone: true}); err == nil {
		t.Fatal("day 2 must stay locked until day 1 is done")
	}
	if _, err := ApplyProgress(a.ID, ProgressPatch{Day: 1, GameDone: true}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildGameHasTappableItems(t *testing.T) {
	for day := 1; day <= 30; day++ {
		game, err := BuildGame(day)
		if err != nil {
			t.Fatalf("day %d: %v", day, err)
		}
		if len(game.Items) == 0 {
			t.Fatalf("day %d has empty game", day)
		}
	}
}
