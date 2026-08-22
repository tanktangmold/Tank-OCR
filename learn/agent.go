package learn

import (
	"strings"
	"unicode"
)

// MatchInterest is the learning agent: it reads what the child likes and
// returns beginner stories (and a book-search query) around that interest.
func MatchInterest(query string) (MatchResult, error) {
	cat, err := LoadCatalog()
	if err != nil {
		return MatchResult{}, err
	}

	q := normalizeQuery(query)
	if q == "" {
		return MatchResult{
			Query:     query,
			MessageJA: "すきなものを教えてね。サッカーでも、えほんでもいいよ。",
			MessageZH: "告诉我你喜欢什么。足球也可以，绘本也可以。",
			Stories:   cat.Stories,
		}, nil
	}

	interestID, specific := classifyInterest(q)
	var picked []Story
	for _, story := range cat.Stories {
		if story.Interest == interestID {
			picked = append(picked, story)
		}
	}

	// C罗 / Ronaldo should lead with the childhood biography story.
	if specific == "ronaldo" {
		picked = prioritizeStory(picked, "ronaldo-childhood")
	}

	if len(picked) == 0 {
		generated := GenerateStory(characterNameFor(q), q)
		picked = []Story{generated}
	}

	bookQuery := bookQueryFor(q, specific, picked)
	msgJA, msgZH := agentMessage(q, specific, interestID, picked)

	return MatchResult{
		Query:     query,
		MessageJA: msgJA,
		MessageZH: msgZH,
		BookQuery: bookQuery,
		Stories:   picked,
	}, nil
}

func normalizeQuery(query string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(query)) {
		if unicode.IsSpace(r) || r == '　' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func classifyInterest(q string) (interestID, specific string) {
	if containsAny(q, ronaldoKeys...) {
		return "football", "ronaldo"
	}
	if containsAny(q, footballKeys...) {
		return "football", "football"
	}
	if containsAny(q, bookKeys...) {
		return "books", "books"
	}
	if containsAny(q, animalKeys...) {
		return "animals", "animals"
	}
	if containsAny(q, spaceKeys...) {
		return "space", "space"
	}
	if DetectActivity(q) != nil {
		return "", "activity"
	}
	return "", ""
}

func prioritizeStory(stories []Story, id string) []Story {
	var first []Story
	var rest []Story
	for _, story := range stories {
		if story.ID == id {
			first = append(first, story)
		} else {
			rest = append(rest, story)
		}
	}
	return append(first, rest...)
}

func bookQueryFor(q, specific string, stories []Story) string {
	if specific == "ronaldo" {
		return "Cristiano Ronaldo Ultimate Football Heroes children biography"
	}
	if specific == "football" {
		return "Cristiano Ronaldo children football picture book"
	}
	if len(stories) > 0 && stories[0].BookQuery != "" {
		return stories[0].BookQuery
	}
	if q != "" {
		return q + " children picture book"
	}
	return "children picture book"
}

func characterNameFor(q string) string {
	if containsAny(q, ronaldoKeys...) {
		return "罗纳尔多"
	}
	return "小明"
}

func agentMessage(q, specific, interestID string, stories []Story) (ja, zh string) {
	switch {
	case specific == "ronaldo":
		return "サッカー選手のCロナウドだね！子どものころから有名になるまでの話を、かんたんな中国語で一文ずつ読めるよ。ネット上の子ども向けの本も探したよ。",
			"是足球运动员C罗！可以从小时候讲到成为球员。每句都很简单。我也去找了相关的儿童书。"
	case interestID == "football":
		return "サッカーがすきなんだね！Cロナウドの子どものころの話と、日本の子どものサッカーの話があるよ。好きな絵本があれば、それに近い中国語の文も作れるよ。",
			"你喜欢足球呀！有C罗小时候的故事，也有住在日本的小朋友踢球的故事。"
	case interestID == "books":
		return "えほんがすきなんだね！一ページ、一文で読める中国語の本の話を用意したよ。好きな本の名前を教えてくれたら、それに近い本も探すよ。",
			"你喜欢绘本呀！可以用一页一句的中文来读。告诉我书名，我也可以去找相近的书。"
	case interestID == "animals":
		return "どうぶつがすきなんだね！ボールであそぶこいぬの、かんたんな中国語の話があるよ。",
			"你喜欢小动物呀！有一只喜欢球的小狗，句子都很短。"
	case interestID == "space":
		return "お星さまがすきなんだね！夜空の、かんたんな中国語の話があるよ。",
			"你喜欢星星呀！有一段很短的夜空故事。"
	case len(stories) > 0 && stories[0].Generated:
		title := stories[0].Title.JA
		return "「" + q + "」がすきなんだね！そのすきなものから、かんたんな中国語の文を作ったよ。絵のスタイルも変えられるよ。　" + title,
			"你喜欢「" + q + "」呀！我用这个爱好写成了很短的中文句子。"
	default:
		return "このすきなものから話を探したよ。気に入ったものを選んでね。",
			"我按这个爱好找了故事。选一个开始吧。"
	}
}

func containsAny(q string, keys ...string) bool {
	for _, key := range keys {
		if strings.Contains(q, normalizeQuery(key)) {
			return true
		}
	}
	return false
}

var (
	ronaldoKeys = []string{
		"ronaldo", "cristiano", "c罗", "c羅", "cロ", "ロナウド", "ロナaldo",
		"クリスティアーノ", "克里斯蒂亚诺", "罗纳尔多", "羅納爾多", "lored",
	}
	footballKeys = []string{
		"football", "soccer", "サッカー", "さっかー", "足球", "蹴球", "フットサル",
		"ボール", "ball",
	}
	bookKeys = []string{
		"book", "絵本", "えほん", "绘本", "繪本", "読書", "读物", "书", "本",
		"児童書", "児童",
	}
	animalKeys = []string{
		"animal", "dog", "puppy", "cat", "どうぶつ", "動物", "动物",
		"いぬ", "犬", "ねこ", "猫", "小狗", "小猫", "わんわん", "旺旺",
	}
	spaceKeys = []string{
		"star", "space", "moon", "宇宙", "うちゅう", "星星", "星", "月", "そら",
	}
)
