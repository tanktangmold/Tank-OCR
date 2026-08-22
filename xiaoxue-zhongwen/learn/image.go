package learn

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	mediaDir    = "data/media"
	staticDir   = "static"
	imageClient = &http.Client{Timeout: 45 * time.Second}
	imageLocks  sync.Map
	imageAPI    = "https://image.pollinations.ai/prompt/"
	imageModel  = "turbo" // cheapest / fastest public model
)

// SetMediaPaths sets cache and fallback image directories.
func SetMediaPaths(media, static string) {
	mediaDir = media
	staticDir = static
}

// SentenceImagePrompt builds a cheap-model prompt that matches this Chinese line.
func SentenceImagePrompt(zh, scene, style string) string {
	look := styleLook(style)
	visual := sentenceVisual(zh, scene)
	return look + " Curly-haired boy, red football shirt, kid-safe. " + visual + ". No text."
}

func styleLook(style string) string {
	switch style {
	case "picturebook":
		return "Cheap children's watercolor picture book illustration, soft pastel, paper texture."
	case "realistic":
		return "Inexpensive gentle painterly children's illustration, natural light, not photographic."
	default:
		return "Cheap kids comic illustration, thick ink outlines, bright flat colors, cel shading."
	}
}

func sentenceVisual(zh, scene string) string {
	var parts []string
	for _, rule := range visualRules {
		if strings.Contains(zh, rule.zh) {
			parts = append(parts, rule.en)
			if len(parts) == 2 {
				break
			}
		}
	}
	if len(parts) == 0 {
		parts = append(parts, sceneVisual(scene))
	}
	parts = append(parts, zh)
	out := strings.Join(parts, ", ")
	if len([]rune(out)) > 160 {
		r := []rune(out)
		out = string(r[:160])
	}
	return out
}

type visualRule struct{ zh, en string }

var visualRules = []visualRule{
	{"小岛", "small green Atlantic island and modest house by the sea"},
	{"葡萄牙", "Portugal coastal village"},
	{"家很小", "tiny crowded home interior"},
	{"家里人", "warm family in a small kitchen"},
	{"爸爸", "father working, kind tired man"},
	{"妈妈", "mother cooking, kind woman"},
	{"钱不多", "simple poor but clean home, no luxury"},
	{"爱", "family hug, warm light"},
	{"三岁", "very small toddler boy with a football"},
	{"哥哥", "older brother playing with the boy"},
	{"旧", "worn old black-and-white soccer ball"},
	{"街上", "narrow village street used as a pitch"},
	{"邻居", "neighbors watching from doorways"},
	{"天黑", "dusk, boy still running with a ball"},
	{"球队", "small local football club, kids in mismatched kits"},
	{"教练", "kind youth football coach"},
	{"训练", "training cones on a dusty pitch"},
	{"汗", "sweat, extra practice after others left"},
	{"第一个", "boy arriving first at the empty pitch at dawn"},
	{"最后", "boy leaving last from the pitch at night"},
	{"十岁", "ten-year-old boy at a bigger football academy"},
	{"十二岁", "twelve-year-old boy with a small backpack leaving home"},
	{"离开", "goodbye at a hillside house, backpack and football"},
	{"里斯本", "Lisbon city, big buildings, homesick boy"},
	{"想家", "boy sitting on a bunk looking at a family photo"},
	{"哭", "boy wiping tears but holding a football"},
	{"口音", "classmates pointing, boy standing brave"},
	{"打电话", "boy on an old telephone talking to mom"},
	{"两只脚", "boy practicing kicking with both feet"},
	{"十五岁", "teenage boy in a hospital sports clinic"},
	{"心跳", "doctor checking a young athlete's heart"},
	{"手术", "small medical room, brave teenage boy, hopeful"},
	{"站起来", "boy standing up again on a grass pitch"},
	{"爱哭", "boy turning sadness into extra training"},
	{"过人", "dribbling past another young player"},
	{"射门", "shooting at a goal"},
	{"进球", "celebrating a goal, kids cheering"},
	{"鼓掌", "crowd clapping in a stadium"},
	{"十八岁", "eighteen-year-old arriving in England with a suitcase and ball"},
	{"英国", "England football stadium and grey sky"},
	{"曼联", "big professional training ground, red training kit, original character"},
	{"有名", "famous footballer waving to kids, still kind"},
	{"忘记", "remembering the small island house"},
	{"帮助", "star player coaching little children on an island"},
	{"梦想", "boy looking at the night sky over the ocean, football under his arm"},
	{"复习", "children looking at picture cards of a football story"},
	{"努力", "hard training, determined smile"},
}

func sceneVisual(scene string) string {
	switch scene {
	case "island", "house":
		return "seaside island home"
	case "firstkick", "kick", "daily", "run", "effort":
		return "boy practicing football"
	case "academy", "goodbye":
		return "youth football school"
	case "star", "travel":
		return "football stadium hope"
	default:
		return "boy with a soccer ball"
	}
}

// ImageCachePath is the on-disk file for a generated sentence picture.
func ImageCachePath(style, prompt string) string {
	sum := sha1.Sum([]byte(style + "|" + prompt))
	name := fmt.Sprintf("%x.jpg", sum[:10])
	return filepath.Join(mediaDir, "images", name)
}

// EnsureSentenceImage returns a JPEG path, generating with the cheap turbo model if needed.
func EnsureSentenceImage(zh, scene, style string) (string, error) {
	if style == "" {
		style = "comic"
	}
	prompt := SentenceImagePrompt(zh, scene, style)
	path := ImageCachePath(style, prompt)
	if st, err := os.Stat(path); err == nil && st.Size() > 1000 {
		return path, nil
	}
	lock := imageLock(path)
	lock.Lock()
	defer lock.Unlock()
	if st, err := os.Stat(path); err == nil && st.Size() > 1000 {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := fetchCheapImage(prompt, path); err != nil {
		fallback := filepath.Join(staticDir, "images", style, fallbackScene(scene)+".webp")
		if _, statErr := os.Stat(fallback); statErr == nil {
			return fallback, nil
		}
		return "", err
	}
	return path, nil
}

func imageLock(path string) *sync.Mutex {
	v, _ := imageLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func fetchCheapImage(prompt, dest string) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 800 * time.Millisecond)
		}
		last = fetchCheapImageOnce(prompt, dest)
		if last == nil {
			return nil
		}
	}
	return last
}

func fetchCheapImageOnce(prompt, dest string) error {
	seed := int(sha1.Sum([]byte(prompt))[0])
	u := imageAPI + url.PathEscape(prompt) + fmt.Sprintf("?width=768&height=576&nologo=true&model=%s&seed=%d", imageModel, seed)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "xiaoxue-zhongwen/1.0")
	resp, err := imageClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image api status %d", resp.StatusCode)
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n < 2000 {
		_ = os.Remove(tmp)
		return fmt.Errorf("image too small (%d bytes)", n)
	}
	return os.Rename(tmp, dest)
}

func fallbackScene(scene string) string {
	alias := map[string]string{
		"house": "island", "kick": "firstkick", "run": "daily",
		"effort": "daily", "goodbye": "academy", "travel": "star",
	}
	if next, ok := alias[scene]; ok {
		return next
	}
	if scene == "" {
		return "intro"
	}
	return scene
}
