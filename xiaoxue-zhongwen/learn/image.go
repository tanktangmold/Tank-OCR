package learn

import (
	"crypto/sha1"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mediaDir   = "data/media"
	staticDir  = "static"
	imageLocks sync.Map
)

// SetMediaPaths sets cache and fallback image directories.
func SetMediaPaths(media, static string) {
	mediaDir = media
	staticDir = static
}

// SentenceImagePrompt builds a stable cache key that follows this Chinese line.
func SentenceImagePrompt(zh, scene, style string) string {
	look := styleLook(style)
	visual := sentenceVisual(zh, scene)
	return look + " Curly-haired boy, red football shirt, kid-safe. " + visual + "."
}

func styleLook(style string) string {
	switch style {
	case "picturebook":
		return "local children's watercolor picture book illustration, soft pastel, paper texture."
	case "realistic":
		return "local gentle painterly children's illustration, natural light, simple shapes."
	default:
		return "local kids comic illustration, thick ink outlines, bright flat colors."
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
	name := fmt.Sprintf("%x.svg", sum[:10])
	return filepath.Join(mediaDir, "images", name)
}

// EnsureSentenceImage returns a local SVG path. It is intentionally network-free:
// a lesson page should never wait on a slow image model before the child can continue.
func EnsureSentenceImage(zh, scene, style string) (string, error) {
	if style == "" {
		style = "comic"
	}
	prompt := SentenceImagePrompt(zh, scene, style)
	path := ImageCachePath(style, prompt)
	if st, err := os.Stat(path); err == nil && st.Size() > 400 {
		return path, nil
	}
	lock := imageLock(path)
	lock.Lock()
	defer lock.Unlock()
	if st, err := os.Stat(path); err == nil && st.Size() > 400 {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(localSentenceSVG(zh, scene, style)), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

// PrewarmCourseImages builds every lesson image before children start tapping.
func PrewarmCourseImages(styles []string) error {
	if len(styles) == 0 {
		styles = []string{"comic", "picturebook", "realistic"}
	}
	course := LoadCourse()
	for _, day := range course.Days {
		for i, beat := range day.Lines {
			for _, style := range styles {
				if _, err := EnsureSentenceImage(beat.Sentence.ZH, beat.Scene, style); err != nil {
					return fmt.Errorf("day %d line %d style %s: %w", day.Day, i+1, style, err)
				}
			}
		}
	}
	return nil
}

func imageLock(path string) *sync.Mutex {
	v, _ := imageLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func localSentenceSVG(zh, scene, style string) string {
	p := svgPalette(style)
	concept := imageConcept(zh, scene)
	seed := sha1.Sum([]byte(zh + "|" + scene + "|" + style))
	boyX := 360 + int(seed[0])%56 - 28
	ballX := 470 + int(seed[1])%88 - 44
	cloudX := 78 + int(seed[2])%95
	title := html.EscapeString(zh)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 768 576" role="img" aria-label="%s">
<defs>
<linearGradient id="sky" x1="0" x2="0" y1="0" y2="1"><stop stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient>
<filter id="soft"><feGaussianBlur stdDeviation="%s"/></filter>
</defs>
<rect width="768" height="576" fill="url(#sky)"/>
%s
<ellipse cx="384" cy="520" rx="330" ry="60" fill="%s" opacity=".9"/>
%s
%s
%s
%s
%s
</svg>`,
		title,
		p.skyTop, p.skyBottom, p.blur,
		svgAtmosphere(cloudX, p),
		p.ground,
		svgScene(concept, p),
		svgBoy(boyX, 250, p),
		svgBall(ballX, 424, p),
		svgStyleOverlay(style, p),
		"",
	)
}

type svgColors struct {
	skyTop, skyBottom string
	ground            string
	line              string
	fill              string
	accent            string
	red               string
	blue              string
	shadow            string
	blur              string
}

func svgPalette(style string) svgColors {
	switch style {
	case "picturebook":
		return svgColors{"#fff3cf", "#d9f0ff", "#cde8b5", "#5c4a32", "#fffaf0", "#f7c96f", "#e66954", "#78a6d8", "#000000", "1.6"}
	case "realistic":
		return svgColors{"#dfe8ef", "#f4efe4", "#98b783", "#2e2a22", "#f2eadc", "#c9a566", "#b94e42", "#667f9b", "#000000", "0.4"}
	default:
		return svgColors{"#c9f0ff", "#fff3b8", "#72d06d", "#2c2419", "#fffdf2", "#ffd34d", "#ef4f3d", "#4aa3ff", "#000000", "0"}
	}
}

func imageConcept(zh, scene string) string {
	switch {
	case strings.Contains(zh, "手术") || strings.Contains(zh, "心跳") || strings.Contains(zh, "医生") || strings.Contains(zh, "医院"):
		return "hospital"
	case strings.Contains(zh, "家里人") || strings.Contains(zh, "爸爸") || strings.Contains(zh, "妈妈") || strings.Contains(zh, "爱"):
		return "family"
	case strings.Contains(zh, "离开") || strings.Contains(zh, "十二岁") || strings.Contains(zh, "打电话") || strings.Contains(zh, "想家") || strings.Contains(zh, "哭"):
		return "goodbye"
	case strings.Contains(zh, "里斯本") || strings.Contains(zh, "学院") || strings.Contains(zh, "口音") || scene == "academy":
		return "academy"
	case strings.Contains(zh, "英国") || strings.Contains(zh, "曼联") || strings.Contains(zh, "鼓掌") || strings.Contains(zh, "有名") || scene == "star":
		return "stadium"
	case strings.Contains(zh, "训练") || strings.Contains(zh, "努力") || strings.Contains(zh, "射门") || strings.Contains(zh, "进球") || strings.Contains(zh, "两只脚"):
		return "training"
	case strings.Contains(zh, "小岛") || strings.Contains(zh, "葡萄牙") || strings.Contains(zh, "家很小") || scene == "island" || scene == "house":
		return "island"
	default:
		return sceneVisual(scene)
	}
}

func svgAtmosphere(x int, p svgColors) string {
	return fmt.Sprintf(`<circle cx="640" cy="92" r="46" fill="%s" opacity=".9"/>
<g fill="#fff" opacity=".82"><ellipse cx="%d" cy="86" rx="54" ry="22"/><ellipse cx="%d" cy="78" rx="34" ry="24"/><ellipse cx="%d" cy="90" rx="42" ry="18"/></g>`, p.accent, x, x+44, x+88)
}

func svgScene(concept string, p svgColors) string {
	switch concept {
	case "hospital":
		return fmt.Sprintf(`<rect x="76" y="270" width="210" height="130" rx="18" fill="%s" stroke="%s" stroke-width="6"/>
<rect x="160" y="298" width="40" height="76" rx="8" fill="%s"/><rect x="142" y="316" width="76" height="40" rx="8" fill="%s"/>
<g transform="translate(232 274)">%s</g>`, p.fill, p.line, p.red, p.red, svgAdult("#eef5ff", p))
	case "family":
		return fmt.Sprintf(`<path d="M72 396h232V252L188 176 72 252z" fill="%s" stroke="%s" stroke-width="6"/>
<rect x="132" y="304" width="56" height="92" rx="8" fill="%s"/><rect x="214" y="288" width="54" height="46" fill="%s"/>
<g transform="translate(98 288)">%s</g><g transform="translate(176 282) scale(.9)">%s</g>`, p.fill, p.line, p.blue, p.accent, svgAdult(p.red, p), svgAdult(p.blue, p))
	case "goodbye":
		return fmt.Sprintf(`<path d="M80 402 C170 338 236 338 310 402" fill="none" stroke="%s" stroke-width="8"/>
<path d="M86 370h160v-92l-80-56-80 56z" fill="%s" stroke="%s" stroke-width="6"/>
<path d="M508 384 l38 36 h-76 z" fill="%s" stroke="%s" stroke-width="5"/>
<rect x="500" y="302" width="36" height="92" rx="12" fill="%s"/>`, p.line, p.fill, p.line, p.accent, p.line, p.blue)
	case "academy":
		return fmt.Sprintf(`<rect x="72" y="244" width="248" height="148" rx="16" fill="%s" stroke="%s" stroke-width="6"/>
<rect x="108" y="286" width="48" height="52" fill="%s"/><rect x="180" y="286" width="48" height="52" fill="%s"/><rect x="252" y="286" width="36" height="106" fill="%s"/>
<path d="M426 384h210M426 384v-86h130" fill="none" stroke="%s" stroke-width="8"/>`, p.fill, p.line, p.blue, p.blue, p.accent, p.line)
	case "stadium":
		return fmt.Sprintf(`<path d="M56 354 C178 240 590 240 712 354 L668 410 C540 342 228 342 100 410z" fill="%s" stroke="%s" stroke-width="6"/>
<path d="M120 334 C250 288 518 288 648 334" fill="none" stroke="%s" stroke-width="10"/>
<path d="M154 408h460v-116" fill="none" stroke="%s" stroke-width="7"/>`, p.blue, p.line, p.accent, p.line)
	case "training", "boy practicing football":
		return fmt.Sprintf(`<path d="M118 408h154M118 408v-104h154v104" fill="none" stroke="%s" stroke-width="8"/>
<path d="M92 444h50l-25-58zM214 444h50l-25-58z" fill="%s" stroke="%s" stroke-width="4"/>
<path d="M530 416 C594 388 650 388 704 416" fill="none" stroke="%s" stroke-width="7"/>`, p.line, p.accent, p.line, p.line)
	default:
		return fmt.Sprintf(`<path d="M68 390 C136 326 228 326 300 390" fill="%s" stroke="%s" stroke-width="6"/>
<path d="M86 386h152v-92l-76-54-76 54z" fill="%s" stroke="%s" stroke-width="6"/>
<rect x="138" y="326" width="48" height="60" rx="8" fill="%s"/>
<path d="M0 410 C110 372 214 430 326 390 C448 348 550 390 768 356 V576 H0z" fill="%s" opacity=".55"/>`, p.ground, p.line, p.fill, p.line, p.blue, p.blue)
	}
}

func svgAdult(clothes string, p svgColors) string {
	return fmt.Sprintf(`<g><circle cx="32" cy="26" r="22" fill="#f4c69b" stroke="%s" stroke-width="4"/>
<path d="M12 72 C16 42 48 42 52 72 L60 120 H4z" fill="%s" stroke="%s" stroke-width="4"/>
<path d="M12 18 C22 0 48 4 56 24 C42 18 28 18 12 18z" fill="#3b2a1c"/></g>`, p.line, clothes, p.line)
}

func svgBoy(x, y int, p svgColors) string {
	return fmt.Sprintf(`<g transform="translate(%d %d)">
<ellipse cx="78" cy="236" rx="74" ry="18" fill="%s" opacity=".16" filter="url(#soft)"/>
<path d="M58 148 l-28 78M104 148 l34 76" stroke="%s" stroke-width="18" stroke-linecap="round"/>
<path d="M56 140 l-38 42M106 140 l48 26" stroke="#f4c69b" stroke-width="16" stroke-linecap="round"/>
<path d="M42 84 h78 l22 72 H20z" fill="%s" stroke="%s" stroke-width="6" stroke-linejoin="round"/>
<circle cx="80" cy="50" r="42" fill="#f4c69b" stroke="%s" stroke-width="6"/>
<path d="M38 38 C44 0 98 -12 122 30 C100 18 82 28 66 18 C58 30 50 34 38 38z" fill="#2b2118"/>
<circle cx="66" cy="52" r="4" fill="%s"/><circle cx="94" cy="52" r="4" fill="%s"/>
<path d="M68 70 Q80 80 96 70" fill="none" stroke="%s" stroke-width="4" stroke-linecap="round"/>
</g>`, x-80, y, p.shadow, p.line, p.red, p.line, p.line, p.line, p.line, p.line)
}

func svgBall(x, y int, p svgColors) string {
	return fmt.Sprintf(`<g transform="translate(%d %d)">
<circle cx="0" cy="0" r="34" fill="#fff" stroke="%s" stroke-width="5"/>
<path d="M0 -20 L18 -6 L12 18 H-12 L-18 -6z" fill="%s"/>
<path d="M0 -20 L0 -34M18 -6 L32 -14M12 18 L24 30M-12 18 L-24 30M-18 -6 L-32 -14" stroke="%s" stroke-width="4"/>
</g>`, x, y, p.line, p.line, p.line)
}

func svgStyleOverlay(style string, p svgColors) string {
	switch style {
	case "picturebook":
		return `<rect width="768" height="576" fill="#fff" opacity=".10"/><g opacity=".16" stroke="#8b704c" stroke-width="1"><path d="M0 82h768M0 186h768M0 302h768M0 438h768"/></g>`
	case "realistic":
		return `<rect width="768" height="576" fill="#000" opacity=".04"/><ellipse cx="384" cy="276" rx="350" ry="242" fill="#fff" opacity=".10"/>`
	default:
		return fmt.Sprintf(`<rect x="18" y="18" width="732" height="540" rx="32" fill="none" stroke="%s" stroke-width="6" opacity=".16"/>`, p.line)
	}
}
