package learn

import (
	"fmt"
	"strings"
)

// DayLesson is one 15-minute lesson in the month-long C罗 course.
type DayLesson struct {
	Day      int    `json:"day"`
	Week     int    `json:"week"`
	Title    Text   `json:"title"`
	Moral    Text   `json:"moral"`
	Scene    string `json:"scene"`
	GameKind string `json:"gameKind"`
	Minutes  int    `json:"minutes"`
	Lines    []Beat `json:"lines"`
}

// Course is the 30-day connected biography.
type Course struct {
	ID      string      `json:"id"`
	Title   Text        `json:"title"`
	Summary Text        `json:"summary"`
	Days    []DayLesson `json:"days"`
}

type rawLine struct {
	zh, ja, patZH, patJA string
}

type rawDay struct {
	day, week        int
	titleZH, titleJA string
	moralZH, moralJA string
	scene            string
	lines            []rawLine
}

func line(zh, ja, patZH, patJA string) rawLine {
	return rawLine{zh: zh, ja: ja, patZH: patZH, patJA: patJA}
}

func buildLine(raw rawLine, scene string) Beat {
	tokens := Tokenize(raw.zh)
	pinyinParts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.Pinyin != "" {
			pinyinParts = append(pinyinParts, token.Pinyin)
		}
	}
	return Beat{
		Scene:       scene,
		ImagePrompt: SentenceImagePrompt(raw.zh, scene, "comic"),
		Pattern: Pattern{
			ZH: raw.patZH,
			JA: raw.patJA,
		},
		Sentence: Sentence{
			ZH:     raw.zh,
			Pinyin: strings.Join(pinyinParts, " "),
			JA:     raw.ja,
			Tokens: tokens,
		},
	}
}

func buildDay(raw rawDay) DayLesson {
	lines := make([]Beat, 0, len(raw.lines))
	for _, rawLine := range raw.lines {
		lines = append(lines, buildLine(rawLine, raw.scene))
	}
	kinds := []string{"listen-pick", "true-false", "build", "listen-pick"}
	return DayLesson{
		Day:      raw.day,
		Week:     raw.week,
		Title:    Text{ZH: raw.titleZH, JA: raw.titleJA},
		Moral:    Text{ZH: raw.moralZH, JA: raw.moralJA},
		Scene:    raw.scene,
		GameKind: kinds[(raw.day-1)%len(kinds)],
		Minutes:  15,
		Lines:    lines,
	}
}

// LoadCourse returns the 30-day C罗 growth course.
func LoadCourse() Course {
	days := make([]DayLesson, 0, len(rawCourse))
	for _, raw := range rawCourse {
		days = append(days, buildDay(raw))
	}
	return Course{
		ID:    "ronaldo-month",
		Title: Text{ZH: "C罗成长故事 · 三十天", JA: "Cロ成長ものがたり · 30日"},
		Summary: Text{
			ZH: "从小岛上的小孩，到有名的足球运动员。每天十五分钟，一天一段真实的励志故事。",
			JA: "小さな島の子どもが、有名な選手になる。毎日15分、ほんとうの励ましの物語。",
		},
		Days: days,
	}
}

// DayByNumber returns a lesson for 1..30.
func DayByNumber(day int) (DayLesson, bool) {
	course := LoadCourse()
	if day < 1 || day > len(course.Days) {
		return DayLesson{}, false
	}
	return course.Days[day-1], true
}

// ValidateCourse checks length, continuity, and tokenization.
func ValidateCourse(course Course) error {
	if len(course.Days) != 30 {
		return fmt.Errorf("want 30 days, got %d", len(course.Days))
	}
	for i, day := range course.Days {
		if day.Day != i+1 {
			return fmt.Errorf("day index %d has Day=%d", i, day.Day)
		}
		if len(day.Lines) < 7 {
			return fmt.Errorf("day %d needs at least 7 sentences", day.Day)
		}
		for j, beat := range day.Lines {
			if err := validateBeat(fmt.Sprintf("day-%d", day.Day), j, beat); err != nil {
				return err
			}
			if strings.TrimSpace(beat.ImagePrompt) == "" {
				return fmt.Errorf("day %d line %d missing image prompt", day.Day, j)
			}
			for _, token := range beat.Sentence.Tokens {
				if token.Pinyin == "" {
					return fmt.Errorf("day %d line %d token %q missing pinyin", day.Day, j, token.ZH)
				}
			}
		}
	}
	return nil
}
