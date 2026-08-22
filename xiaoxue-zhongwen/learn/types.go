package learn

// Text is a Japanese + Chinese pair used throughout the catalog.
type Text struct {
	ZH string `json:"zh"`
	JA string `json:"ja"`
}

// Token is one tappable Chinese word in a beginner sentence.
type Token struct {
	ZH     string `json:"zh"`
	Pinyin string `json:"pinyin"`
	JA     string `json:"ja"`
}

// PatternPart maps a grammar slot onto sentence tokens.
type PatternPart struct {
	ZH        string `json:"zh"`
	JA        string `json:"ja"`
	TokenFrom int    `json:"tokenFrom"`
	TokenTo   int    `json:"tokenTo"`
}

// Pattern is the grammar frame shown above each example sentence.
type Pattern struct {
	ZH    string        `json:"zh"`
	JA    string        `json:"ja"`
	Parts []PatternPart `json:"parts"`
}

// Sentence is one simple Chinese line with pinyin, Japanese, and word tokens.
type Sentence struct {
	ZH     string  `json:"zh"`
	Pinyin string  `json:"pinyin"`
	JA     string  `json:"ja"`
	Tokens []Token `json:"tokens"`
}

// Beat is one story step: picture + pattern + one sentence.
type Beat struct {
	Scene    string   `json:"scene"`
	Pattern  Pattern  `json:"pattern"`
	Sentence Sentence `json:"sentence"`
}

// Character is the person (or animal) the story is about.
type Character struct {
	ZH         string `json:"zh"`
	JA         string `json:"ja"`
	AlsoCalled string `json:"alsoCalled"`
}

// Story is a linked sequence of beginner sentences.
type Story struct {
	ID         string    `json:"id"`
	Interest   string    `json:"interest"`
	BookQuery  string    `json:"bookQuery"`
	Title      Text      `json:"title"`
	Summary    Text      `json:"summary"`
	Character  Character `json:"character"`
	Beats      []Beat    `json:"beats"`
	Generated  bool      `json:"generated,omitempty"`
	SourceHint string    `json:"sourceHint,omitempty"`
}

// Style is a picture look the child can switch.
type Style struct {
	ID string `json:"id"`
	ZH string `json:"zh"`
	JA string `json:"ja"`
	EN string `json:"en"`
}

// Interest is a chip on the home screen.
type Interest struct {
	ID    string `json:"id"`
	ZH    string `json:"zh"`
	JA    string `json:"ja"`
	Emoji string `json:"emoji"`
}

// Catalog is the full learning library.
type Catalog struct {
	Styles    []Style    `json:"styles"`
	Interests []Interest `json:"interests"`
	Stories   []Story    `json:"stories"`
}

// MatchResult is what the interest agent returns.
type MatchResult struct {
	Query     string  `json:"query"`
	MessageJA string  `json:"messageJa"`
	MessageZH string  `json:"messageZh"`
	BookQuery string  `json:"bookQuery"`
	Stories   []Story `json:"stories"`
}

// Book is a related title the agent found (never copied as lesson text).
type Book struct {
	Title    string `json:"title"`
	Authors  string `json:"authors"`
	Year     int    `json:"year,omitempty"`
	CoverURL string `json:"coverUrl,omitempty"`
	InfoURL  string `json:"infoUrl,omitempty"`
	Source   string `json:"source"`
	NoteJA   string `json:"noteJa"`
	NoteZH   string `json:"noteZh"`
	Curated  bool   `json:"curated"`
}

// BookSearchResult is the agent's book-finding response.
type BookSearchResult struct {
	Query   string `json:"query"`
	Books   []Book `json:"books"`
	Offline bool   `json:"offline"`
	NoteJA  string `json:"noteJa"`
	NoteZH  string `json:"noteZh"`
}
