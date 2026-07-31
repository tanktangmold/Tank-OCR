package handler

import (
	"reflect"
	"testing"
)

func TestParsePageRange(t *testing.T) {
	pages, err := parsePageRange("1-3, 5, 3")
	if err != nil {
		t.Fatalf("parsePageRange returned an error: %v", err)
	}

	expected := []int{1, 2, 3, 5}
	if !reflect.DeepEqual(pages, expected) {
		t.Fatalf("pages = %v, expected %v", pages, expected)
	}
}

func TestParsePageRangeRejectsInvalidInput(t *testing.T) {
	invalid := []string{"0", "3-1", "1-2-3", "abc"}
	for _, value := range invalid {
		if _, err := parsePageRange(value); err == nil {
			t.Errorf("parsePageRange(%q) did not return an error", value)
		}
	}
}

func TestParsePageRangeAllowsEmptySelection(t *testing.T) {
	pages, err := parsePageRange(" ")
	if err != nil {
		t.Fatalf("parsePageRange returned an error: %v", err)
	}
	if pages != nil {
		t.Fatalf("pages = %v, expected nil", pages)
	}
}
