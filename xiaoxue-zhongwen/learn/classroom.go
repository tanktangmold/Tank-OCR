package learn

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Child is one learner profile. Several children can use the same server at once.
type Child struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	CreatedAt string `json:"createdAt"`
}

// DayProgress is one child's work on one calendar day of the course.
type DayProgress struct {
	Heard      []int `json:"heard"`
	LessonDone bool  `json:"lessonDone"`
	GameDone   bool  `json:"gameDone"`
	Seconds    int   `json:"seconds"`
}

// ChildProgress is independent progress for one child.
type ChildProgress struct {
	ChildID       string                  `json:"childId"`
	CurrentDay    int                     `json:"currentDay"`
	CompletedDays []int                   `json:"completedDays"`
	Days          map[string]*DayProgress `json:"days"`
}

// Classroom is the shared multi-child save file.
type Classroom struct {
	Children []Child                   `json:"children"`
	Progress map[string]*ChildProgress `json:"progress"`
}

var (
	classroomMu   sync.Mutex
	classroomPath = "data/classroom.json"
)

// SetClassroomPath sets where child profiles are stored.
func SetClassroomPath(path string) {
	classroomMu.Lock()
	defer classroomMu.Unlock()
	classroomPath = path
}

func emptyClassroom() Classroom {
	return Classroom{
		Children: []Child{},
		Progress: map[string]*ChildProgress{},
	}
}

func loadClassroomLocked() (Classroom, error) {
	data, err := os.ReadFile(classroomPath)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyClassroom(), nil
		}
		return Classroom{}, err
	}
	var room Classroom
	if err := json.Unmarshal(data, &room); err != nil {
		return Classroom{}, err
	}
	if room.Progress == nil {
		room.Progress = map[string]*ChildProgress{}
	}
	return room, nil
}

func saveClassroomLocked(room Classroom) error {
	if err := os.MkdirAll(filepath.Dir(classroomPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(room, "", "  ")
	if err != nil {
		return err
	}
	tmp := classroomPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, classroomPath)
}

// ListChildren returns every profile.
func ListChildren() ([]Child, error) {
	classroomMu.Lock()
	defer classroomMu.Unlock()
	room, err := loadClassroomLocked()
	if err != nil {
		return nil, err
	}
	return room.Children, nil
}

// AddChild creates a profile another iPad can also pick.
func AddChild(name, avatar string) (Child, error) {
	name = trimName(name)
	if name == "" {
		return Child{}, fmt.Errorf("name required")
	}
	if avatar == "" {
		avatar = "⚽"
	}
	child := Child{
		ID:        fmt.Sprintf("c%x", time.Now().UnixNano()),
		Name:      name,
		Avatar:    avatar,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	classroomMu.Lock()
	defer classroomMu.Unlock()
	room, err := loadClassroomLocked()
	if err != nil {
		return Child{}, err
	}
	room.Children = append(room.Children, child)
	room.Progress[child.ID] = &ChildProgress{
		ChildID:    child.ID,
		CurrentDay: 1,
		Days:       map[string]*DayProgress{},
	}
	if err := saveClassroomLocked(room); err != nil {
		return Child{}, err
	}
	return child, nil
}

// GetProgress returns one child's independent course progress.
func GetProgress(childID string) (ChildProgress, error) {
	classroomMu.Lock()
	defer classroomMu.Unlock()
	room, err := loadClassroomLocked()
	if err != nil {
		return ChildProgress{}, err
	}
	progress, ok := room.Progress[childID]
	if !ok {
		return ChildProgress{}, fmt.Errorf("child not found")
	}
	if progress.Days == nil {
		progress.Days = map[string]*DayProgress{}
	}
	if progress.CurrentDay < 1 {
		progress.CurrentDay = 1
	}
	return *progress, nil
}

// ProgressPatch is a partial update from a lesson or game session.
type ProgressPatch struct {
	Day        int   `json:"day"`
	Heard      []int `json:"heard"`
	LessonDone bool  `json:"lessonDone"`
	GameDone   bool  `json:"gameDone"`
	Seconds    int   `json:"seconds"`
}

// ApplyProgress updates one child without touching other children's data.
func ApplyProgress(childID string, patch ProgressPatch) (ChildProgress, error) {
	if patch.Day < 1 || patch.Day > 30 {
		return ChildProgress{}, fmt.Errorf("day must be 1-30")
	}
	classroomMu.Lock()
	defer classroomMu.Unlock()
	room, err := loadClassroomLocked()
	if err != nil {
		return ChildProgress{}, err
	}
	progress, ok := room.Progress[childID]
	if !ok {
		return ChildProgress{}, fmt.Errorf("child not found")
	}
	if progress.Days == nil {
		progress.Days = map[string]*DayProgress{}
	}
	if !dayUnlocked(progress, patch.Day) {
		return ChildProgress{}, fmt.Errorf("day %d is locked", patch.Day)
	}
	key := fmt.Sprintf("%d", patch.Day)
	dayState := progress.Days[key]
	if dayState == nil {
		dayState = &DayProgress{}
		progress.Days[key] = dayState
	}
	if len(patch.Heard) > 0 {
		dayState.Heard = uniqueInts(append(dayState.Heard, patch.Heard...))
	}
	if patch.Seconds > dayState.Seconds {
		dayState.Seconds = patch.Seconds
	}
	if patch.LessonDone {
		dayState.LessonDone = true
		progress.CompletedDays = uniqueInts(append(progress.CompletedDays, patch.Day))
		if progress.CurrentDay < patch.Day+1 && patch.Day < 30 {
			progress.CurrentDay = patch.Day + 1
		}
		if patch.Day == 30 {
			progress.CurrentDay = 30
		}
	}
	if patch.GameDone {
		if !dayState.LessonDone {
			return ChildProgress{}, fmt.Errorf("finish today's 15-minute lesson first")
		}
		dayState.GameDone = true
	}
	if err := saveClassroomLocked(room); err != nil {
		return ChildProgress{}, err
	}
	return *progress, nil
}

func dayUnlocked(progress *ChildProgress, day int) bool {
	if day <= 1 {
		return true
	}
	prev := progress.Days[fmt.Sprintf("%d", day-1)]
	return prev != nil && prev.LessonDone
}

func trimName(name string) string {
	runes := []rune(name)
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		out = append(out, r)
		if len(out) >= 12 {
			break
		}
	}
	return string(out)
}

func uniqueInts(values []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// LANURLs lists addresses other iPads can open on the same Wi-Fi.
func LANURLs(port string) []string {
	var urls []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return urls
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		urls = append(urls, "http://"+ipNet.IP.String()+":"+port)
	}
	return urls
}

// ChildExists reports whether a profile id is known.
func ChildExists(childID string) bool {
	classroomMu.Lock()
	defer classroomMu.Unlock()
	room, err := loadClassroomLocked()
	if err != nil {
		return false
	}
	_, ok := room.Progress[childID]
	return ok
}
