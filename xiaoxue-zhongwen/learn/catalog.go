package learn

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed stories.json
var storiesJSON []byte

var (
	catalogOnce sync.Once
	catalogVal  Catalog
	catalogErr  error
)

// LoadCatalog parses the embedded beginner-story library.
func LoadCatalog() (Catalog, error) {
	catalogOnce.Do(func() {
		dec := json.NewDecoder(bytes.NewReader(storiesJSON))
		dec.DisallowUnknownFields()
		catalogErr = dec.Decode(&catalogVal)
	})
	return catalogVal, catalogErr
}

// StoryByID returns a catalog story or false if it is not present.
func StoryByID(id string) (Story, bool) {
	cat, err := LoadCatalog()
	if err != nil {
		return Story{}, false
	}
	for _, story := range cat.Stories {
		if story.ID == id {
			return story, true
		}
	}
	return Story{}, false
}

// ValidateCatalog checks that every sentence's tokens cover the Chinese text.
func ValidateCatalog(cat Catalog) error {
	if len(cat.Stories) == 0 {
		return fmt.Errorf("catalog has no stories")
	}
	for _, story := range cat.Stories {
		if story.ID == "" || len(story.Beats) == 0 {
			return fmt.Errorf("story %q is incomplete", story.ID)
		}
		for i, beat := range story.Beats {
			if err := validateBeat(story.ID, i, beat); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBeat(storyID string, index int, beat Beat) error {
	if beat.Sentence.ZH == "" || len(beat.Sentence.Tokens) == 0 {
		return fmt.Errorf("%s beat %d: missing sentence tokens", storyID, index)
	}
	joined := ""
	for _, token := range beat.Sentence.Tokens {
		if token.ZH == "" {
			return fmt.Errorf("%s beat %d: empty token", storyID, index)
		}
		joined += token.ZH
	}
	want := stripSentencePunct(beat.Sentence.ZH)
	if joined != want {
		return fmt.Errorf("%s beat %d: tokens %q != sentence %q", storyID, index, joined, want)
	}
	return nil
}

func stripSentencePunct(s string) string {
	cutset := "。！？.!?、，"
	return trimRightRunes(s, cutset)
}

func trimRightRunes(s, cutset string) string {
	for len(s) > 0 {
		r := []rune(s)
		last := r[len(r)-1]
		if !containsRune(cutset, last) {
			break
		}
		s = string(r[:len(r)-1])
	}
	return s
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
