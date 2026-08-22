package learn

import (
	"sort"
	"strings"
	"unicode/utf8"
)

type dictWord struct {
	ZH     string
	Pinyin string
	JA     string
}

var wordList = []dictWord{
	{"罗纳尔多", "Luónà'ěrduō", "ロナウド"},
	{"乒乓球", "pīngpāngqiú", "卓球"},
	{"足球", "zúqiú", "サッカー"},
	{"篮球", "lánqiú", "バスケットボール"},
	{"游泳", "yóuyǒng", "スイミング"},
	{"小朋友", "xiǎo péngyou", "こども"},
	{"十八岁", "shíbā suì", "18さい"},
	{"三岁", "sān suì", "3さい"},
	{"四岁", "sì suì", "4さい"},
	{"五岁", "wǔ suì", "5さい"},
	{"每天", "měi tiān", "まいにち"},
	{"开始", "kāishǐ", "はじめる"},
	{"喜欢", "xǐhuan", "すき"},
	{"高兴", "gāoxìng", "うれしい"},
	{"小明", "Xiǎomíng", "ミン"},
	{"日本", "Rìběn", "にほん"},
	{"英国", "Yīngguó", "イギリス"},
	{"学校", "xuéxiào", "がっこう"},
	{"明星", "míngxīng", "スター"},
	{"朋友", "péngyou", "ともだち"},
	{"踢球", "tīqiú", "ボールをける"},
	{"打球", "dǎqiú", "ボールをうつ"},
	{"打乒乓球", "dǎ pīngpāngqiú", "卓球をする"},
	{"踢足球", "tī zúqiú", "サッカーをする"},
	{"打篮球", "dǎ lánqiú", "バスケをする"},
	{"中文", "Zhōngwén", "中国語"},
	{"星星", "xīngxing", "ほし"},
	{"小孩", "xiǎohái", "こども"},
	{"天上", "tiān shàng", "そらのうえ"},
	{"看书", "kàn shū", "本をよむ"},
	{"好书", "hǎo shū", "いい本"},
	{"一直", "yīzhí", "ずっと"},
	{"努力", "nǔlì", "がんばる"},
	{"离开", "líkāi", "はなれる"},
	{"放学后", "fàngxué hòu", "ほうかご"},
	{"开心", "kāixīn", "うれしい"},
	{"旺旺", "Wàngwang", "ワンワン"},
	{"这", "zhè", "これ"},
	{"是", "shì", "～です"},
	{"他", "tā", "かれ"},
	{"她", "tā", "かのじょ"},
	{"我", "wǒ", "わたし"},
	{"也", "yě", "も"},
	{"很", "hěn", "とても"},
	{"的", "de", "の"},
	{"了", "le", "した"},
	{"在", "zài", "に"},
	{"去", "qù", "いく"},
	{"有", "yǒu", "ある"},
	{"看", "kàn", "みる"},
	{"学", "xué", "まなぶ"},
	{"住", "zhù", "すむ"},
	{"跑", "pǎo", "はしる"},
	{"踢", "tī", "ける"},
	{"打", "dǎ", "うつ"},
	{"游", "yóu", "およぐ"},
	{"玩", "wán", "あそぶ"},
	{"让", "ràng", "～させる"},
	{"都", "dōu", "いつも"},
	{"得", "de", "～のようす"},
	{"快", "kuài", "はやい"},
	{"小", "xiǎo", "ちいさい"},
	{"大", "dà", "おおきい"},
	{"好", "hǎo", "よい"},
	{"家", "jiā", "いえ"},
	{"书", "shū", "ほん"},
	{"球", "qiú", "ボール"},
	{"狗", "gǒu", "いぬ"},
	{"岛", "dǎo", "しま"},
	{"上", "shàng", "のうえ"},
	{"里", "lǐ", "のなか"},
	{"成", "chéng", "なる"},
	{"亮", "liàng", "あかるい"},
	{"它", "tā", "それ"},
	{"一", "yī", "いち"},
	{"个", "gè", "こ"},
	{"本", "běn", "さつ"},
	{"天", "tiān", "ひ"},
}

var sortedWords []dictWord

func init() {
	sortedWords = append([]dictWord(nil), wordList...)
	sort.Slice(sortedWords, func(i, j int) bool {
		li := utf8.RuneCountInString(sortedWords[i].ZH)
		lj := utf8.RuneCountInString(sortedWords[j].ZH)
		if li == lj {
			return sortedWords[i].ZH > sortedWords[j].ZH
		}
		return li > lj
	})
}

func lookupWord(zh string) Token {
	for _, w := range wordList {
		if w.ZH == zh {
			return Token{ZH: w.ZH, Pinyin: w.Pinyin, JA: w.JA}
		}
	}
	return Token{}
}

func lookupMust(zh string) Token {
	if tok := lookupWord(zh); tok.ZH != "" {
		return tok
	}
	return Token{ZH: zh, Pinyin: "", JA: zh}
}

// Tokenize splits a beginner sentence with longest-word matching.
func Tokenize(sentence string) []Token {
	s := stripSentencePunct(strings.TrimSpace(sentence))
	var out []Token
	for len(s) > 0 {
		matched := false
		for _, w := range sortedWords {
			if strings.HasPrefix(s, w.ZH) {
				out = append(out, Token{ZH: w.ZH, Pinyin: w.Pinyin, JA: w.JA})
				s = s[len(w.ZH):]
				matched = true
				break
			}
		}
		if !matched {
			r, size := utf8.DecodeRuneInString(s)
			out = append(out, Token{ZH: string(r), Pinyin: "", JA: ""})
			s = s[size:]
		}
	}
	return out
}
