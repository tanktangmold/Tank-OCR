package learn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestTTSCachePathDiffersByVoice(t *testing.T) {
	a := TTSCachePath("你好", "xiaoxiao")
	b := TTSCachePath("你好", "yunxi")
	c := TTSCachePath("谢谢", "xiaoxiao")
	if a == b {
		t.Fatal("male and female voices must not share an mp3")
	}
	if a == c {
		t.Fatal("different text must not share an mp3")
	}
}

func TestEnsureTTSUsesCachedFile(t *testing.T) {
	prevMedia, prevStatic := mediaDir, staticDir
	t.Cleanup(func() { SetMediaPaths(prevMedia, prevStatic) })
	dir := t.TempDir()
	SetMediaPaths(dir, "static")
	path := TTSCachePath("你好", "xiaoxiao")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fake-mp3-bytes-for-cache-hit-test----"), 0o644); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := EnsureTTS("你好", "xiaoxiao")
			if err != nil {
				errs <- err
				return
			}
			if got != path {
				errs <- fmt.Errorf("path %s != %s", got, path)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
