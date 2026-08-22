package learn

import (
	"strings"
	"testing"
)

func TestSentenceImagePromptDiffersByLine(t *testing.T) {
	a := SentenceImagePrompt("他三岁开始踢足球。", "firstkick", "comic")
	b := SentenceImagePrompt("他住在小岛上。", "island", "comic")
	if a == b {
		t.Fatal("different sentences should get different pictures")
	}
	if !strings.Contains(a, "三岁") && !strings.Contains(a, "toddler") {
		t.Fatalf("prompt should follow the sentence: %s", a)
	}
}

func TestVoicesIncludeFemaleAndMale(t *testing.T) {
	var female, male int
	for _, voice := range Voices {
		if voice.Gender == "female" {
			female++
		}
		if voice.Gender == "male" {
			male++
		}
	}
	if female < 2 || male < 2 {
		t.Fatalf("female=%d male=%d", female, male)
	}
	if VoiceByID("yunyang").Gender != "male" {
		t.Fatal("yunyang should be male")
	}
	if VoiceByID("xiaoxiao").Gender != "female" {
		t.Fatal("xiaoxiao should be female")
	}
}
