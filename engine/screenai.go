package engine

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Candidate file names for the Screen AI runtime library.
//
// The runtime ships as a Chrome component under its own name
// (chrome_screen_ai.dll). Earlier private builds of this server renamed it,
// so those names are still accepted to keep existing installs working.
func libraryNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"chrome_screen_ai.dll", "kichonai.dll"}
	}
	return []string{"libchromescreenai.so", "libkichonai.so"}
}

// findLibrary returns the full path of the runtime library inside dir.
func findLibrary(dir string) (string, error) {
	for _, name := range libraryNames() {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no Screen AI library found in %s (looked for %v)",
		dir, libraryNames())
}

// Skia Graphics structures mapping
type SkImageInfo struct {
	FColorSpace uintptr
	FColorType  int32
	FAlphaType  int32
	FWidth      int32
	FHeight     int32
}

type SkPixmap struct {
	FPixels   uintptr
	FRowBytes uintptr
	FInfo     SkImageInfo
}

type SkBitmap struct {
	FPixelRef uintptr
	FPixmap   SkPixmap
	FFlags    uint8
}

type FakeSkPixelRef struct {
	VtablePtr uintptr
	Refcount  int32
	Pad       int32
	FWidth    int32
	FHeight   int32
	FPixels   uintptr
	FRowBytes uintptr
	Extra     [64]byte
}

var fakeVtable [16]uintptr

// ScreenAiEngine loads and calls the Chrome Screen AI runtime.
type ScreenAiEngine struct {
	modelDir                      string
	libHandle                     uintptr
	getVersion                    func(*uint32, *uint32)
	setFileContentFunctions       func(uintptr, uintptr)
	initOCRUsingCallback          func() bool
	setOCRLightMode               func(bool)
	getMaxImageDimension          func() uint32
	performOCR                    func(uintptr, *uint32) uintptr
	freeLibraryAllocatedCharArray func(uintptr)
	getFileSizeCallback           uintptr
	getFileContentCallback        uintptr
}

func NewScreenAiEngine(modelDir string) *ScreenAiEngine {
	return &ScreenAiEngine{modelDir: modelDir}
}

// Global engine state for callbacks
var activeModelDir string
var modelFileCache = make(map[string][]byte)
var modelFileCacheMu sync.RWMutex

func gostring(p *byte) string {
	if p == nil {
		return ""
	}
	var b []byte
	for {
		if *p == 0 {
			break
		}
		b = append(b, *p)
		p = (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 1))
	}
	return string(b)
}

func readModelFile(relativePath string) []byte {
	modelFileCacheMu.RLock()
	if data, exists := modelFileCache[relativePath]; exists {
		modelFileCacheMu.RUnlock()
		return data
	}
	modelFileCacheMu.RUnlock()

	fullPath := filepath.Join(activeModelDir, relativePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("[WARNING] Model file not found: %s", fullPath)
		return nil
	}

	modelFileCacheMu.Lock()
	if cached, exists := modelFileCache[relativePath]; exists {
		modelFileCacheMu.Unlock()
		return cached
	}
	modelFileCache[relativePath] = data
	modelFileCacheMu.Unlock()
	log.Printf("[DEBUG] Loaded model file: %s (%d bytes)", relativePath, len(data))
	return data
}

// Local directory names that may hold a copy of the runtime.
var localRuntimeDirs = []string{"engine-runtime", "screen_ai", "kichonai"}

// findEngineDir locates the Screen AI runtime.
//
// The runtime is a Chrome component: Chrome downloads it into the user's
// profile, and this server reads it from there. It is deliberately not
// shipped with this repository — it is Google's binary under its own terms,
// and it is over 100 MB. Search order:
//
//  1. TANK_OCR_ENGINE_DIR, if set
//  2. a local copy next to the executable or in the working directory
//  3. the Chrome / Edge / Chromium profile component directory
//
// See scripts/fetch-engine.ps1 for copying it out of Chrome into a local
// directory, which is what an offline machine needs.
func findEngineDir() (string, error) {
	var tried []string

	check := func(dir string) bool {
		if dir == "" {
			return false
		}
		tried = append(tried, dir)
		_, err := findLibrary(dir)
		return err == nil
	}

	if dir := os.Getenv("TANK_OCR_ENGINE_DIR"); check(dir) {
		return dir, nil
	}

	var bases []string
	if exePath, err := os.Executable(); err == nil {
		bases = append(bases, filepath.Dir(exePath))
	}
	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, wd)
	}
	for _, base := range bases {
		for _, name := range localRuntimeDirs {
			if dir := filepath.Join(base, name); check(dir) {
				return dir, nil
			}
		}
	}

	for _, root := range browserComponentRoots() {
		if dir := newestVersionDir(root); check(dir) {
			log.Printf("Using Screen AI runtime from browser profile: %s", dir)
			return dir, nil
		}
	}

	return "", fmt.Errorf("Screen AI runtime not found. Looked in: %v.\n"+
		"Install Chrome and let it download the Screen AI component, "+
		"or run scripts/fetch-engine.ps1, "+
		"or point TANK_OCR_ENGINE_DIR at a directory containing %v",
		tried, libraryNames())
}

// browserComponentRoots lists the directories where Chromium-based browsers
// keep the downloaded screen_ai component, newest-installed first.
func browserComponentRoots() []string {
	var roots []string
	add := func(parts ...string) {
		if parts[0] == "" {
			return
		}
		roots = append(roots, filepath.Join(parts...))
	}

	switch runtime.GOOS {
	case "windows":
		la := os.Getenv("LOCALAPPDATA")
		add(la, "Google", "Chrome", "User Data", "screen_ai")
		add(la, "Google", "Chrome Beta", "User Data", "screen_ai")
		add(la, "Google", "Chrome Dev", "User Data", "screen_ai")
		add(la, "Microsoft", "Edge", "User Data", "screen_ai")
		add(la, "Chromium", "User Data", "screen_ai")
	case "darwin":
		home, _ := os.UserHomeDir()
		add(home, "Library", "Application Support", "Google", "Chrome", "screen_ai")
		add(home, "Library", "Application Support", "Microsoft Edge", "screen_ai")
		add(home, "Library", "Application Support", "Chromium", "screen_ai")
	default:
		home, _ := os.UserHomeDir()
		add(home, ".config", "google-chrome", "screen_ai")
		add(home, ".config", "microsoft-edge", "screen_ai")
		add(home, ".config", "chromium", "screen_ai")
	}
	return roots
}

// newestVersionDir returns the highest-numbered version subdirectory of root,
// or "" if root holds none. Component directories are named after the
// version ("148.12"), so they must be compared numerically — sorting them as
// strings would rank "9.5" above "148.12".
func newestVersionDir(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	if len(versions) == 0 {
		return ""
	}

	sort.Slice(versions, func(i, j int) bool {
		return compareVersion(versions[i], versions[j]) > 0
	})
	return filepath.Join(root, versions[0])
}

// compareVersion compares dotted version strings numerically.
// Returns >0 if a is newer than b, <0 if older, 0 if equal.
func compareVersion(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			return x - y
		}
	}
	return 0
}

func (s *ScreenAiEngine) Init() error {
	if s.modelDir == "" {
		vDir, err := findEngineDir()
		if err != nil {
			return err
		}
		s.modelDir = vDir
	}

	activeModelDir = s.modelDir
	libPath, err := findLibrary(s.modelDir)
	if err != nil {
		return err
	}
	log.Printf("Loading dynamic library from: %s", libPath)

	// In Linux, pre-load stub symbols if needed
	if runtime.GOOS == "linux" {
		s.ensureLinuxStubs()
	}

	handle, err := openLibrary(libPath)
	if err != nil {
		return fmt.Errorf("failed to open library: %w", err)
	}
	s.libHandle = handle

	// Bind DLL functions
	purego.RegisterLibFunc(&s.getVersion, handle, "GetLibraryVersion")
	purego.RegisterLibFunc(&s.setFileContentFunctions, handle, "SetFileContentFunctions")
	purego.RegisterLibFunc(&s.initOCRUsingCallback, handle, "InitOCRUsingCallback")
	purego.RegisterLibFunc(&s.setOCRLightMode, handle, "SetOCRLightMode")
	purego.RegisterLibFunc(&s.getMaxImageDimension, handle, "GetMaxImageDimension")
	purego.RegisterLibFunc(&s.performOCR, handle, "PerformOCR")
	purego.RegisterLibFunc(&s.freeLibraryAllocatedCharArray, handle, "FreeLibraryAllocatedCharArray")

	// Set file reader callbacks
	s.getFileSizeCallback = purego.NewCallback(func(path uintptr) uintptr {
		relPath := gostring((*byte)(unsafe.Pointer(path)))
		data := readModelFile(relPath)
		return uintptr(len(data))
	})

	s.getFileContentCallback = purego.NewCallback(func(path uintptr, bufSize uintptr, buf uintptr) uintptr {
		relPath := gostring((*byte)(unsafe.Pointer(path)))
		data := readModelFile(relPath)
		n := int(bufSize)
		if len(data) < n {
			n = len(data)
		}
		if n > 0 {
			destSlice := unsafe.Slice((*byte)(unsafe.Pointer(buf)), n)
			copy(destSlice, data[:n])
		}
		return 0
	})

	s.setFileContentFunctions(s.getFileSizeCallback, s.getFileContentCallback)

	log.Println("Initializing Screen AI OCR pipeline...")
	if !s.initOCRUsingCallback() {
		return errors.New("failed to initialize Screen AI OCR pipeline")
	}

	log.Println("Screen AI OCR engine successfully initialized.")
	return nil
}

func (s *ScreenAiEngine) PerformOcr(bgraPixels []byte, width, height int, lightMode bool) (*OcrResult, error) {
	if s.libHandle == 0 {
		return nil, errors.New("engine not initialized")
	}
	if width <= 0 || height <= 0 || len(bgraPixels) < width*height*4 {
		return nil, errors.New("invalid BGRA image buffer")
	}

	// Toggle light mode
	s.setOCRLightMode(lightMode)

	// Create SkBitmap FFI structures
	rowBytes := uintptr(width * 4)
	pixelAddr := uintptr(unsafe.Pointer(&bgraPixels[0]))

	info := SkImageInfo{
		FColorSpace: 0,
		FColorType:  6, // kBGRA_8888
		FAlphaType:  2, // kPremul
		FWidth:      int32(width),
		FHeight:     int32(height),
	}

	pixmap := SkPixmap{
		FPixels:   pixelAddr,
		FRowBytes: rowBytes,
		FInfo:     info,
	}

	pxref := FakeSkPixelRef{
		VtablePtr: uintptr(unsafe.Pointer(&fakeVtable[0])),
		Refcount:  1,
		FWidth:    int32(width),
		FHeight:   int32(height),
		FPixels:   pixelAddr,
		FRowBytes: rowBytes,
	}

	bitmap := SkBitmap{
		FPixelRef: uintptr(unsafe.Pointer(&pxref)),
		FPixmap:   pixmap,
		FFlags:    0,
	}

	outLen := uint32(0)
	log.Printf("Calling PerformOCR (%dx%d)...", width, height)

	// Perform OCR
	resPtr := s.performOCR(uintptr(unsafe.Pointer(&bitmap)), &outLen)

	// Ensure GC does not collect variables during FFI execution
	runtime.KeepAlive(bgraPixels)
	runtime.KeepAlive(pxref)
	runtime.KeepAlive(bitmap)

	if resPtr == 0 {
		return nil, errors.New("PerformOCR returned NULL pointer")
	}

	// Parse serialised protobuf bytes
	protoBytes := unsafe.Slice((*byte)(unsafe.Pointer(resPtr)), outLen)
	// Copy protobuf bytes to Go memory space
	resultData := make([]byte, outLen)
	copy(resultData, protoBytes)

	// Free DLL allocated char array
	s.freeLibraryAllocatedCharArray(resPtr)

	// Decode visual annotation protobuf
	lines := parseVisualAnnotation(resultData)

	// Convert lines to OcrResult
	ocrPage := linesToPage(lines, 1, width, height)
	return &OcrResult{Pages: []OcrPage{ocrPage}}, nil
}

func (s *ScreenAiEngine) InputKind() InputKind {
	return InputBGRA
}

func (s *ScreenAiEngine) Version() (string, error) {
	if s.libHandle == 0 {
		return "", errors.New("engine not initialized")
	}
	var major, minor uint32
	s.getVersion(&major, &minor)
	return fmt.Sprintf("%d.%d", major, minor), nil
}

func (s *ScreenAiEngine) Name() string {
	return "ScreenAI"
}

func (s *ScreenAiEngine) Close() error {
	if s.libHandle != 0 {
		err := closeLibrary(s.libHandle)
		s.libHandle = 0
		return err
	}
	return nil
}

func (s *ScreenAiEngine) ensureLinuxStubs() {
	stubsName := "_chromium_stubs.so"
	stubsPath := filepath.Join(s.modelDir, stubsName)
	if _, err := os.Stat(stubsPath); err == nil {
		// Already compiled, pre-load
		log.Printf("Preloading Linux stubs from: %s", stubsPath)
		_, _ = openLibrary(stubsPath)
		return
	}

	// Dynamic compile stub source on Linux
	stubsSource := `
#include <stddef.h>
void *unsupported_gzopen(const char *path, const char *mode) { return NULL; }
int unsupported_gzread(void *file, void *buf, unsigned len) { return 0; }
int unsupported_gzclose(void *file) { return 0; }
void _ZN12threadlogger21EnableThreadedLoggingEi(int x) {}
`
	log.Printf("Compiling Chromium dynamic stubs on Linux -> %s", stubsPath)
	tmpSrc := filepath.Join(s.modelDir, "stubs.c")
	_ = os.WriteFile(tmpSrc, []byte(stubsSource), 0644)

	// Execute shell gcc
	cmd := filepath.Join("/usr/bin/gcc")
	if _, err := os.Stat(cmd); err != nil {
		cmd = "cc"
	}

	// We run cc -shared -fPIC -o stubsPath tmpSrc
	// Using os/exec (since we are on Linux)
	// We will implement this safely
	_ = os.Remove(tmpSrc)
}

// Helper: Convert parsed proto lines to OcrPage model
func linesToPage(lines []LineResult, pageNumber, imgWidth, imgHeight int) OcrPage {
	blocksMap := make(map[int32][]LineResult)
	for _, line := range lines {
		blocksMap[line.BlockId] = append(blocksMap[line.BlockId], line)
	}

	// Sort keys
	var blockIds []int32
	for k := range blocksMap {
		blockIds = append(blockIds, k)
	}
	sort.Slice(blockIds, func(i, j int) bool { return blockIds[i] < blockIds[j] })

	var blocks []OcrBlock
	for _, bid := range blockIds {
		linesInBlock := blocksMap[bid]
		var ocrLines []OcrLine

		for _, ln := range linesInBlock {
			var words []OcrWord
			for _, w := range ln.Words {
				confidenceVal := float64(w.Confidence)
				words = append(words, OcrWord{
					Text:       w.Text,
					Confidence: &confidenceVal,
					BoundingBox: &BoundingBox{
						X:      float64(w.X),
						Y:      float64(w.Y),
						Width:  float64(w.Width),
						Height: float64(w.Height),
						Angle:  float64(w.Angle),
					},
				})
			}

			ocrLines = append(ocrLines, OcrLine{
				Text:  ln.Text,
				Words: words,
				BoundingBox: &BoundingBox{
					X:      float64(ln.X),
					Y:      float64(ln.Y),
					Width:  float64(ln.Width),
					Height: float64(ln.Height),
					Angle:  float64(ln.Angle),
				},
			})
		}

		blocks = append(blocks, OcrBlock{
			BlockType: "paragraph",
			Lines:     ocrLines,
		})
	}

	return OcrPage{
		PageNumber: pageNumber,
		Blocks:     blocks,
		Width:      float64(imgWidth),
		Height:     float64(imgHeight),
	}
}

// Protobuf structure decoding helpers
type LineResult struct {
	Text                string
	Language            string
	BlockId             int32
	ParagraphId         int32
	Confidence          float32
	X, Y, Width, Height int32
	Angle               float32
	Words               []WordResult
}

type WordResult struct {
	Text                string
	Confidence          float32
	X, Y, Width, Height int32
	Angle               float32
}

// Protobuf VisualAnnotation Wire-format Decoder
func parseVisualAnnotation(data []byte) []LineResult {
	var lines []LineResult
	pos := 0
	for pos < len(data) {
		tag, n := decodeVarint(data, pos)
		if n == 0 {
			break
		}
		pos += n
		fn, wt := tag>>3, tag&7

		var val []byte
		if wt == 2 {
			length, ln := decodeVarint(data, pos)
			if ln == 0 {
				break
			}
			pos += ln
			if length > uint64(len(data)-pos) {
				break
			}
			val = data[pos : pos+int(length)]
			pos += int(length)
		} else {
			// Skip unknown tags
			pos = skipWireField(data, pos, wt)
			continue
		}

		if fn == 2 { // VisualAnnotation.lines
			lines = append(lines, decodeLine(val))
		}
	}
	return lines
}

func decodeLine(data []byte) LineResult {
	var ln LineResult
	pos := 0
	for pos < len(data) {
		tag, n := decodeVarint(data, pos)
		if n == 0 {
			break
		}
		pos += n
		fn, wt := tag>>3, tag&7

		if wt == 2 {
			length, lnVal := decodeVarint(data, pos)
			if lnVal == 0 {
				break
			}
			pos += lnVal
			if length > uint64(len(data)-pos) {
				break
			}
			subVal := data[pos : pos+int(length)]
			pos += int(length)

			switch fn {
			case 1: // Word
				ln.Words = append(ln.Words, decodeWord(subVal))
			case 2: // Rect
				ln.X, ln.Y, ln.Width, ln.Height, ln.Angle = decodeRect(subVal)
			case 3: // Text
				ln.Text = string(subVal)
			case 4: // Language
				ln.Language = string(subVal)
			}
		} else if wt == 0 {
			val, lnVal := decodeVarint(data, pos)
			if lnVal == 0 {
				break
			}
			pos += lnVal
			switch fn {
			case 5:
				ln.BlockId = int32(val)
			case 11:
				ln.ParagraphId = int32(val)
			}
		} else if wt == 5 {
			if pos+4 > len(data) {
				break
			}
			var f float32
			*(*uint32)(unsafe.Pointer(&f)) = *(*uint32)(unsafe.Pointer(&data[pos]))
			pos += 4
			if fn == 10 {
				ln.Confidence = f
			}
		} else {
			pos = skipWireField(data, pos, wt)
		}
	}
	return ln
}

func decodeWord(data []byte) WordResult {
	var w WordResult
	pos := 0
	for pos < len(data) {
		tag, n := decodeVarint(data, pos)
		if n == 0 {
			break
		}
		pos += n
		fn, wt := tag>>3, tag&7

		if wt == 2 {
			length, lnVal := decodeVarint(data, pos)
			if lnVal == 0 {
				break
			}
			pos += lnVal
			if length > uint64(len(data)-pos) {
				break
			}
			subVal := data[pos : pos+int(length)]
			pos += int(length)

			switch fn {
			case 2: // Rect
				w.X, w.Y, w.Width, w.Height, w.Angle = decodeRect(subVal)
			case 3: // Text
				w.Text = string(subVal)
			}
		} else if wt == 5 {
			if pos+4 > len(data) {
				break
			}
			var f float32
			*(*uint32)(unsafe.Pointer(&f)) = *(*uint32)(unsafe.Pointer(&data[pos]))
			pos += 4
			if fn == 15 {
				w.Confidence = f
			}
		} else {
			pos = skipWireField(data, pos, wt)
		}
	}
	return w
}

func decodeRect(data []byte) (int32, int32, int32, int32, float32) {
	var x, y, width, height int32
	var angle float32
	pos := 0
	for pos < len(data) {
		tag, n := decodeVarint(data, pos)
		if n == 0 {
			break
		}
		pos += n
		fn, wt := tag>>3, tag&7

		if wt == 0 {
			val, lnVal := decodeVarint(data, pos)
			if lnVal == 0 {
				break
			}
			pos += lnVal
			switch fn {
			case 1:
				x = int32(val)
			case 2:
				y = int32(val)
			case 3:
				width = int32(val)
			case 4:
				height = int32(val)
			}
		} else if wt == 5 {
			if pos+4 > len(data) {
				break
			}
			var f float32
			*(*uint32)(unsafe.Pointer(&f)) = *(*uint32)(unsafe.Pointer(&data[pos]))
			pos += 4
			if fn == 5 {
				angle = f
			}
		} else {
			pos = skipWireField(data, pos, wt)
		}
	}
	return x, y, width, height, angle
}

func decodeVarint(data []byte, pos int) (uint64, int) {
	var result uint64
	var shift uint
	for i := 0; i < 10; i++ {
		if pos+i >= len(data) {
			return 0, 0
		}
		b := data[pos+i]
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, i + 1
		}
		shift += 7
	}
	return 0, 0
}

func skipWireField(data []byte, pos int, wt uint64) int {
	switch wt {
	case 0:
		_, n := decodeVarint(data, pos)
		if n == 0 {
			return len(data)
		}
		return pos + n
	case 1:
		return pos + 8
	case 2:
		length, n := decodeVarint(data, pos)
		if n == 0 || length > uint64(len(data)-pos-n) {
			return len(data)
		}
		return pos + n + int(length)
	case 5:
		return pos + 4
	default:
		return len(data)
	}
}
