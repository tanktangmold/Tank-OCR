package learn

import (
	"strings"
	"unicode/utf8"
)

// Activity is a sport or hobby used to fill the beginner sentence patterns.
type Activity struct {
	ID        string
	Keys      []string
	NounZH    string
	NounJA    string
	VerbZH    string
	VerbJA    string
	PhraseZH  string
	PhraseJA  string
	Scene     string
	AgeZH     string
	AgeJA     string
	AgePinyin string
}

var activities = []Activity{
	{
		ID: "football", Keys: []string{"football", "soccer", "サッカー", "さっかー", "足球", "フットサル"},
		NounZH: "足球", NounJA: "サッカー", VerbZH: "踢", VerbJA: "ける",
		PhraseZH: "踢足球", PhraseJA: "サッカーをする", Scene: "firstkick",
		AgeZH: "三岁", AgeJA: "3さい", AgePinyin: "sān suì",
	},
	{
		ID: "pingpong", Keys: []string{"pingpong", "ping-pong", "tabletennis", "卓球", "乒乓球", "ピンポン"},
		NounZH: "乒乓球", NounJA: "卓球", VerbZH: "打", VerbJA: "うつ",
		PhraseZH: "打乒乓球", PhraseJA: "卓球をする", Scene: "firstkick",
		AgeZH: "三岁", AgeJA: "3さい", AgePinyin: "sān suì",
	},
	{
		ID: "basketball", Keys: []string{"basketball", "バスケ", "バスケット", "篮球", "籃球"},
		NounZH: "篮球", NounJA: "バスケットボール", VerbZH: "打", VerbJA: "うつ",
		PhraseZH: "打篮球", PhraseJA: "バスケをする", Scene: "park",
		AgeZH: "五岁", AgeJA: "5さい", AgePinyin: "wǔ suì",
	},
	{
		ID: "swim", Keys: []string{"swim", "swimming", "水泳", "泳ぐ", "游泳"},
		NounZH: "泳", NounJA: "スイミング", VerbZH: "游", VerbJA: "およぐ",
		PhraseZH: "游泳", PhraseJA: "およぐ", Scene: "park",
		AgeZH: "四岁", AgeJA: "4さい", AgePinyin: "sì suì",
	},
}

// DetectActivity maps free-text interest onto a known hobby, if any.
func DetectActivity(q string) *Activity {
	n := normalizeQuery(q)
	for i := range activities {
		act := &activities[i]
		for _, key := range act.Keys {
			if strings.Contains(n, normalizeQuery(key)) {
				return act
			}
		}
	}
	return nil
}

// GenerateStory writes an original beginner chain from a name + interest.
// It uses the same "pattern + one simple sentence" shape as the curated stories.
func GenerateStory(name, interest string) Story {
	if name == "" {
		name = "小明"
	}
	act := DetectActivity(interest)
	nounZH, nounJA := "球", "ボール"
	phraseZH, phraseJA := "玩", "あそぶ"
	verbZH := "玩"
	scene := "park"
	ageZH, ageJA, agePinyin := "三岁", "3さい", "sān suì"
	if act != nil {
		nounZH, nounJA = act.NounZH, act.NounJA
		phraseZH, phraseJA = act.PhraseZH, act.PhraseJA
		verbZH = act.VerbZH
		scene = act.Scene
		ageZH, ageJA, agePinyin = act.AgeZH, act.AgeJA, act.AgePinyin
	} else {
		nounZH = shortInterestLabel(interest)
		nounJA = interest
		phraseZH = "玩"
		phraseJA = interest + "であそぶ"
	}

	nameTok := lookupWord(name)
	if nameTok.ZH == "" {
		nameTok = Token{ZH: name, Pinyin: "", JA: name}
	}

	beats := []Beat{
		makeBeat("intro", "这是 + 谁", "これは + だれ",
			"这是"+name+"。",
			"Zhè shì "+orPinyin(nameTok)+".",
			"これは"+nameTok.JA+"です。",
			[]Token{lookupMust("这"), lookupMust("是"), nameTok}),
		makeBeat(scene, "谁 + 喜欢 + 什么", "だれ + すき + なに",
			name+"喜欢"+nounZH+"。",
			orPinyin(nameTok)+" xǐhuan "+lookupMust(nounZH).Pinyin+".",
			nameTok.JA+"は"+nounJA+"がすきです。",
			[]Token{nameTok, lookupMust("喜欢"), lookupMust(nounZH)}),
		makeBeat(scene, "谁 + 几岁 + 开始 + 做什么", "だれ + なんさい + はじめる + すること",
			name+ageZH+"开始"+phraseZH+"。",
			orPinyin(nameTok)+" "+agePinyin+" kāishǐ "+lookupMust(phraseZH).Pinyin+".",
			nameTok.JA+"は"+ageJA+"で"+phraseJA+"を始めました。",
			joinTokens([]Token{nameTok, lookupMust(ageZH), lookupMust("开始")}, phraseTokens(verbZH, nounZH, phraseZH))),
		makeBeat("daily", "谁 + 每天 + 都 + 做什么", "だれ + まいにち + いつも + すること",
			name+"每天都玩。",
			orPinyin(nameTok)+" měi tiān dōu wán.",
			nameTok.JA+"はまいにち遊びます。",
			[]Token{nameTok, lookupMust("每天"), lookupMust("都"), lookupMust("玩")}),
		makeBeat("star", "谁 + 很 + 怎么样", "だれ + とても + どんな",
			name+"很高兴。",
			orPinyin(nameTok)+" hěn gāoxìng.",
			nameTok.JA+"はとてもうれしいです。",
			[]Token{nameTok, lookupMust("很"), lookupMust("高兴")}),
		makeBeat("book", "谁 + 也 + 喜欢 + 什么", "だれ + も + すき + なに",
			"我也喜欢"+nounZH+"。",
			"Wǒ yě xǐhuan "+lookupMust(nounZH).Pinyin+".",
			"わたしも"+nounJA+"がすきです。",
			[]Token{lookupMust("我"), lookupMust("也"), lookupMust("喜欢"), lookupMust(nounZH)}),
	}

	for i := range beats {
		if err := validateBeat("generated", i, beats[i]); err != nil {
			beats[i].Sentence.Tokens = Tokenize(beats[i].Sentence.ZH)
		}
	}

	titleNoun := nounJA
	if act != nil {
		titleNoun = act.NounJA
	}
	return Story{
		ID:         "generated-" + slug(interest),
		Interest:   "generated",
		BookQuery:  strings.TrimSpace(interest + " children picture book"),
		Title:      Text{ZH: name + "喜欢" + nounZH, JA: nameTok.JA + "の" + titleNoun},
		Summary:    Text{ZH: "一句一句，跟着爱好学中文。", JA: "すきなものから、一文ずつ中国語を学ぶよ。"},
		Character:  Character{ZH: name, JA: nameTok.JA, AlsoCalled: name},
		Beats:      beats,
		Generated:  true,
		SourceHint: "original-pattern",
	}
}

func makeBeat(scene, patZH, patJA, zh, pinyin, ja string, tokens []Token) Beat {
	return Beat{
		Scene: scene,
		Pattern: Pattern{
			ZH: patZH,
			JA: patJA,
		},
		Sentence: Sentence{
			ZH:     zh,
			Pinyin: pinyin,
			JA:     ja,
			Tokens: tokens,
		},
	}
}

func phraseTokens(verbZH, nounZH, phraseZH string) []Token {
	if tok := lookupWord(phraseZH); tok.ZH != "" {
		return []Token{tok}
	}
	out := make([]Token, 0, 2)
	if verbZH != "" {
		if tok := lookupWord(verbZH); tok.ZH != "" {
			out = append(out, tok)
		}
	}
	if tok := lookupWord(nounZH); tok.ZH != "" {
		out = append(out, tok)
	}
	if len(out) == 0 {
		return Tokenize(phraseZH + "。")
	}
	return out
}

func joinTokens(parts ...[]Token) []Token {
	var out []Token
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func orPinyin(tok Token) string {
	if tok.Pinyin != "" {
		return tok.Pinyin
	}
	return tok.ZH
}

func shortInterestLabel(interest string) string {
	n := strings.TrimSpace(interest)
	if utf8.RuneCountInString(n) > 4 {
		r := []rune(n)
		return string(r[:4])
	}
	if n == "" {
		return "球"
	}
	return n
}

func slug(s string) string {
	s = normalizeQuery(s)
	if s == "" {
		return "story"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r > 127 {
			b.WriteRune(r)
		}
		if b.Len() > 24 {
			break
		}
	}
	if b.Len() == 0 {
		return "story"
	}
	return b.String()
}
