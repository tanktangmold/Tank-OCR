package learn

import "testing"

func TestCourseIsThirtyConnectedDays(t *testing.T) {
	course := LoadCourse()
	if err := ValidateCourse(course); err != nil {
		t.Fatal(err)
	}
	if course.Days[0].Lines[0].Sentence.ZH != "这是罗纳尔多。" {
		t.Fatalf("story should start with C罗: %s", course.Days[0].Lines[0].Sentence.ZH)
	}
	last := course.Days[29].Lines
	found := false
	for _, beat := range last {
		if beat.Sentence.ZH == "他没有放弃。" || beat.Sentence.ZH == "努力最重要。" {
			found = true
		}
	}
	if !found {
		t.Fatal("month should end on effort / not giving up")
	}
}
