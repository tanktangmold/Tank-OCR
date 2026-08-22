package learn

import (
	"strings"
	"testing"
)

func TestCatalogTokensCoverSentences(t *testing.T) {
	cat, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCatalog(cat); err != nil {
		t.Fatal(err)
	}
}

func TestMatchFootballLeadsWithRonaldo(t *testing.T) {
	result, err := MatchInterest("サッカーがすき")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stories) == 0 {
		t.Fatal("expected football stories")
	}
	if result.Stories[0].ID != "ronaldo-childhood" && result.Stories[0].Interest != "football" {
		t.Fatalf("unexpected first story %s", result.Stories[0].ID)
	}
	found := false
	for _, story := range result.Stories {
		if story.ID == "ronaldo-childhood" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected C罗 childhood story")
	}
	if !strings.Contains(result.BookQuery, "Ronaldo") && !strings.Contains(result.BookQuery, "football") {
		t.Fatalf("book query = %q", result.BookQuery)
	}
}

func TestMatchRonaldoAliases(t *testing.T) {
	for _, q := range []string{"C罗", "ロナウド", "Ronaldo C", "Lored C"} {
		result, err := MatchInterest(q)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Stories) == 0 || result.Stories[0].ID != "ronaldo-childhood" {
			t.Fatalf("query %q first story = %#v", q, result.Stories)
		}
	}
}

func TestMatchBooksAndAnimals(t *testing.T) {
	books, err := MatchInterest("えほん")
	if err != nil {
		t.Fatal(err)
	}
	if len(books.Stories) == 0 || books.Stories[0].Interest != "books" {
		t.Fatalf("books match = %#v", books.Stories)
	}
	animals, err := MatchInterest("いぬ")
	if err != nil {
		t.Fatal(err)
	}
	if len(animals.Stories) == 0 || animals.Stories[0].Interest != "animals" {
		t.Fatalf("animals match = %#v", animals.Stories)
	}
}

func TestGeneratePingPongUsesAgePattern(t *testing.T) {
	result, err := MatchInterest("卓球")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stories) != 1 || !result.Stories[0].Generated {
		t.Fatalf("expected generated ping-pong story, got %#v", result.Stories)
	}
	found := false
	for _, beat := range result.Stories[0].Beats {
		if strings.Contains(beat.Sentence.ZH, "三岁开始打乒乓球") {
			found = true
			if err := validateBeat(result.Stories[0].ID, 0, beat); err != nil {
				t.Fatal(err)
			}
		}
		if strings.Contains(beat.Pattern.ZH, "几岁") && !strings.Contains(beat.Pattern.ZH, "开始") {
			t.Fatalf("age pattern should include 开始: %s", beat.Pattern.ZH)
		}
	}
	if !found {
		t.Fatalf("missing 三岁开始打乒乓球 in %#v", result.Stories[0].Beats)
	}
}

func TestTokenizeLongestMatch(t *testing.T) {
	tokens := Tokenize("他三岁开始踢足球。")
	joined := ""
	for _, token := range tokens {
		joined += token.ZH
	}
	if joined != "他三岁开始踢足球" {
		t.Fatalf("joined = %q tokens=%#v", joined, tokens)
	}
	if len(tokens) < 4 {
		t.Fatalf("too few tokens: %#v", tokens)
	}
}

func TestCuratedRonaldoBooks(t *testing.T) {
	result := SearchBooks("Cristiano Ronaldo")
	if len(result.Books) == 0 {
		t.Fatal("expected curated books even without network")
	}
	found := false
	for _, book := range result.Books {
		if strings.Contains(book.Title, "Ronaldo") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a Ronaldo title in %#v", result.Books)
	}
}
