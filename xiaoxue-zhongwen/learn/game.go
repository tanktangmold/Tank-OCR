package learn

import "strings"

// Game is one reward game unlocked after a 15-minute lesson.
type Game struct {
	Day   int        `json:"day"`
	Kind  string     `json:"kind"`
	Title Text       `json:"title"`
	Hint  Text       `json:"hint"`
	Items []GameItem `json:"items"`
}

// GameItem is one tappable question.
type GameItem struct {
	Prompt    Text     `json:"prompt"`
	AudioZH   string   `json:"audioZh"`
	Options   []string `json:"options"`
	Answer    int      `json:"answer"`
	Tokens    []Token  `json:"tokens,omitempty"`
	TrueFalse *bool    `json:"trueFalse,omitempty"`
}

var skipGameWords = map[string]bool{
	"他": true, "的": true, "了": true, "很": true, "在": true, "是": true,
	"也": true, "都": true, "不": true, "有": true, "和": true, "我": true,
	"你": true, "这": true, "还": true, "更": true, "就": true, "又": true,
	"要": true, "会": true, "把": true, "给": true, "从": true, "到": true,
}

// BuildGame makes a football-themed practice game from that day's sentences.
func BuildGame(day int) (Game, error) {
	lesson, ok := DayByNumber(day)
	if !ok {
		return Game{}, errNoDay(day)
	}
	switch lesson.GameKind {
	case "true-false":
		return buildTrueFalse(lesson), nil
	case "build":
		return buildSentenceGame(lesson), nil
	default:
		return buildListenPick(lesson), nil
	}
}

func errNoDay(day int) error {
	return errDay(day)
}

type errDay int

func (e errDay) Error() string {
	return "day not found"
}

func buildListenPick(lesson DayLesson) Game {
	words := contentWords(lesson, 8)
	items := make([]GameItem, 0, len(words))
	all := allContentWords()
	for i, word := range words {
		options := []string{word.ZH}
		for _, extra := range all {
			if extra.ZH == word.ZH {
				continue
			}
			options = append(options, extra.ZH)
			if len(options) == 4 {
				break
			}
		}
		for len(options) < 4 {
			options = append(options, options[0])
		}
		rotate := i % 4
		options = rotateOptions(options, rotate)
		answer := 0
		for idx, opt := range options {
			if opt == word.ZH {
				answer = idx
				break
			}
		}
		items = append(items, GameItem{
			Prompt:  Text{ZH: "听一听，选一选", JA: "きいて、えらんでね"},
			AudioZH: word.ZH,
			Options: options,
			Answer:  answer,
		})
	}
	return Game{
		Day:   lesson.Day,
		Kind:  "listen-pick",
		Title: Text{ZH: "射门选词", JA: "シュートことば"},
		Hint:  Text{ZH: "点开声音，再点正确的中文。", JA: "音声を聞いて、ただしい中国語をおしてね。"},
		Items: items,
	}
}

func buildTrueFalse(lesson DayLesson) Game {
	items := make([]GameItem, 0, 6)
	for i, beat := range lesson.Lines {
		if i >= 6 {
			break
		}
		truth := true
		promptZH := beat.Sentence.ZH
		promptJA := "この文は今日の話と同じ？"
		if i%2 == 1 {
			truth = false
			promptZH = twistSentence(beat.Sentence.ZH)
			promptJA = "この文は今日の話と同じ？"
		}
		tf := truth
		items = append(items, GameItem{
			Prompt:    Text{ZH: promptZH, JA: promptJA},
			AudioZH:   promptZH,
			Options:   []string{"对", "不对"},
			Answer:    boolIndex(truth),
			TrueFalse: &tf,
		})
	}
	return Game{
		Day:   lesson.Day,
		Kind:  "true-false",
		Title: Text{ZH: "对还是不对", JA: "あたるか はずれか"},
		Hint:  Text{ZH: "这是今天的故事吗？", JA: "きょうの物語と同じかな？"},
		Items: items,
	}
}

func buildSentenceGame(lesson DayLesson) Game {
	items := make([]GameItem, 0, 3)
	for i := 0; i < len(lesson.Lines) && len(items) < 3; i++ {
		beat := lesson.Lines[len(lesson.Lines)-1-i]
		if len(beat.Sentence.Tokens) < 3 {
			continue
		}
		options := make([]string, 0, len(beat.Sentence.Tokens))
		for _, token := range beat.Sentence.Tokens {
			options = append(options, token.ZH)
		}
		items = append(items, GameItem{
			Prompt:  Text{ZH: "排一排", JA: "ことばをならべてね"},
			AudioZH: beat.Sentence.ZH,
			Options: rotateOptions(append([]string{}, options...), i+1),
			Answer:  0,
			Tokens:  beat.Sentence.Tokens,
		})
	}
	return Game{
		Day:   lesson.Day,
		Kind:  "build",
		Title: Text{ZH: "组句进球", JA: "ぶんをつくってゴール"},
		Hint:  Text{ZH: "按正确的顺序点汉字。", JA: "ただしいじゅんばんではんじをおしてね。"},
		Items: items,
	}
}

func contentWords(lesson DayLesson, limit int) []Token {
	seen := map[string]bool{}
	var out []Token
	for _, beat := range lesson.Lines {
		for _, token := range beat.Sentence.Tokens {
			if skipGameWords[token.ZH] || seen[token.ZH] || len([]rune(token.ZH)) < 1 {
				continue
			}
			if token.Pinyin == "" {
				continue
			}
			seen[token.ZH] = true
			out = append(out, token)
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func allContentWords() []Token {
	course := LoadCourse()
	seen := map[string]bool{}
	var out []Token
	for _, day := range course.Days {
		for _, token := range contentWords(day, 20) {
			if seen[token.ZH] {
				continue
			}
			seen[token.ZH] = true
			out = append(out, token)
		}
	}
	return out
}

func rotateOptions(options []string, n int) []string {
	if len(options) == 0 {
		return options
	}
	n = n % len(options)
	return append(options[n:], options[:n]...)
}

func boolIndex(v bool) int {
	if v {
		return 0
	}
	return 1
}

func twistSentence(zh string) string {
	repl := [][2]string{
		{"三岁", "八十岁"},
		{"足球", "乒乓球"},
		{"努力", "放弃"},
		{"小岛", "月亮"},
		{"英国", "日本"},
		{"不放弃", "放弃了"},
	}
	for _, pair := range repl {
		if strings.Contains(zh, pair[0]) {
			return strings.Replace(zh, pair[0], pair[1], 1)
		}
	}
	return "他不喜欢足球。"
}
