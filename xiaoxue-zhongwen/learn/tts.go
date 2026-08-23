package learn

import (
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Voice is a Chinese reading voice the child can pick.
type Voice struct {
	ID     string  `json:"id"`
	NameZH string  `json:"nameZh"`
	NameJA string  `json:"nameJa"`
	Gender string  `json:"gender"`
	Edge   string  `json:"-"`
	Pitch  float64 `json:"pitch"` // web-speech fallback
	Rate   float64 `json:"rate"`
}

// Voices is the set of female and male Mandarin voices.
var Voices = []Voice{
	{ID: "xiaoxiao", NameZH: "晓晓", NameJA: "女声・お姉さん", Gender: "female", Edge: "zh-CN-XiaoxiaoNeural", Pitch: 1.12, Rate: 0.9},
	{ID: "xiaoyi", NameZH: "晓伊", NameJA: "女声・やさしい", Gender: "female", Edge: "zh-CN-XiaoyiNeural", Pitch: 1.18, Rate: 0.88},
	{ID: "xiaohan", NameZH: "晓涵", NameJA: "女声・先生", Gender: "female", Edge: "zh-CN-XiaohanNeural", Pitch: 1.05, Rate: 0.86},
	{ID: "yunxi", NameZH: "云希", NameJA: "男声・少年", Gender: "male", Edge: "zh-CN-YunxiNeural", Pitch: 0.88, Rate: 0.92},
	{ID: "yunyang", NameZH: "云扬", NameJA: "男声・お兄さん", Gender: "male", Edge: "zh-CN-YunyangNeural", Pitch: 0.72, Rate: 0.88},
	{ID: "yunjian", NameZH: "云健", NameJA: "男声・先生", Gender: "male", Edge: "zh-CN-YunjianNeural", Pitch: 0.62, Rate: 0.85},
}

var (
	ttsLocks sync.Map
	ttsSlot  = make(chan struct{}, 1) // one edge-tts process at a time
)

func ttsLock(path string) *sync.Mutex {
	v, _ := ttsLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// VoiceByID returns a catalog voice or the default girl voice.
func VoiceByID(id string) Voice {
	for _, voice := range Voices {
		if voice.ID == id {
			return voice
		}
	}
	return Voices[0]
}

// TTSCachePath is the on-disk mp3 for this text and voice.
func TTSCachePath(text, voiceID string) string {
	voice := VoiceByID(voiceID)
	sum := sha1.Sum([]byte(voice.ID + "|" + strings.TrimSpace(text)))
	return filepath.Join(mediaDir, "tts", fmt.Sprintf("%s-%x.mp3", voice.ID, sum[:8]))
}

func ttsReady(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 200
}

// EnsureTTS writes an mp3 of text in the chosen Chinese voice.
func EnsureTTS(text, voiceID string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("empty text")
	}
	path := TTSCachePath(text, voiceID)
	if ttsReady(path) {
		return path, nil
	}
	lock := ttsLock(path)
	lock.Lock()
	defer lock.Unlock()
	if ttsReady(path) {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp.mp3"
	if err := runEdgeTTS(text, VoiceByID(voiceID).Edge, tmp); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

func runEdgeTTS(text, edgeVoice, dest string) error {
	ttsSlot <- struct{}{}
	defer func() { <-ttsSlot }()

	script := `
import asyncio, sys, edge_tts
text, voice, path = sys.argv[1], sys.argv[2], sys.argv[3]
asyncio.run(edge_tts.Communicate(text, voice).save(path))
`
	cmd := exec.Command("python3", "-c", script, text, edgeVoice, dest)
	cmd.Env = os.Environ()
	home, _ := os.UserHomeDir()
	cmd.Env = append(cmd.Env, "PATH="+home+"/.local/bin:"+os.Getenv("PATH"))
	cmd.WaitDelay = 30 * time.Second
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("edge-tts: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
