package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"tank-ocr/engine"

	"github.com/ledongthuc/pdf"
)

type pdfTextPage struct {
	PageNumber int
	Text       string
}

type renderedPage struct {
	PageNumber int
	Path       string
}

func readPDFTextPages(fileBytes []byte, selectedPages map[int]struct{}) ([]pdfTextPage, int, error) {
	reader := bytes.NewReader(fileBytes)
	document, err := pdf.NewReader(reader, int64(len(fileBytes)))
	if err != nil {
		return nil, 0, err
	}

	totalPages := document.NumPage()
	for pageNumber := range selectedPages {
		if pageNumber > totalPages {
			return nil, totalPages, fmt.Errorf("page %d exceeds PDF page count %d", pageNumber, totalPages)
		}
	}

	pages := make([]pdfTextPage, 0, totalPages)
	for pageNumber := 1; pageNumber <= totalPages; pageNumber++ {
		if len(selectedPages) > 0 {
			if _, selected := selectedPages[pageNumber]; !selected {
				continue
			}
		}

		text, err := document.Page(pageNumber).GetPlainText(nil)
		if err != nil {
			return nil, totalPages, err
		}
		pages = append(pages, pdfTextPage{PageNumber: pageNumber, Text: text})
	}

	return pages, totalPages, nil
}

func nativeTextResult(pages []pdfTextPage) (*engine.OcrResult, string, bool) {
	result := &engine.OcrResult{}
	allPagesHaveText := len(pages) > 0

	for _, page := range pages {
		text := strings.TrimSpace(page.Text)
		if text == "" {
			allPagesHaveText = false
		}
		result.Pages = append(result.Pages, engine.OcrPage{
			PageNumber: page.PageNumber,
			Blocks: []engine.OcrBlock{{
				BlockType: "paragraph",
				Lines:     []engine.OcrLine{{Text: page.Text}},
			}},
		})
	}

	return result, flattenText(result), allPagesHaveText
}

func parsePageRange(pageString string) ([]int, error) {
	pageString = strings.TrimSpace(pageString)
	if pageString == "" {
		return nil, nil
	}

	unique := make(map[int]struct{})
	for _, part := range strings.Split(pageString, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid page range %q", part)
			}
			start, errStart := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, errEnd := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if errStart != nil || errEnd != nil || start < 1 || end < start {
				return nil, fmt.Errorf("invalid page range %q", part)
			}
			if end-start > 10000 {
				return nil, fmt.Errorf("page range %q is too large", part)
			}
			for pageNumber := start; pageNumber <= end; pageNumber++ {
				unique[pageNumber] = struct{}{}
			}
			continue
		}

		pageNumber, err := strconv.Atoi(part)
		if err != nil || pageNumber < 1 {
			return nil, fmt.Errorf("invalid page number %q", part)
		}
		unique[pageNumber] = struct{}{}
	}

	pages := make([]int, 0, len(unique))
	for pageNumber := range unique {
		pages = append(pages, pageNumber)
	}
	sort.Ints(pages)
	return pages, nil
}

func toPageSet(pages []int) map[int]struct{} {
	selected := make(map[int]struct{}, len(pages))
	for _, pageNumber := range pages {
		selected[pageNumber] = struct{}{}
	}
	return selected
}

func runPDFToPPM(arguments ...string) error {
	pdftoppmPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		return fmt.Errorf("pdftoppm was not found; install Poppler or add pdftoppm to PATH")
	}

	command := exec.Command(pdftoppmPath, arguments...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pdftoppm failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func renderPDFPages(pdfPath, tempDir string, selectedPages []int) ([]renderedPage, error) {
	renderWithPoppler := func() error {
		if len(selectedPages) == 0 {
			prefix := filepath.Join(tempDir, "page")
			return runPDFToPPM("-png", "-r", "150", pdfPath, prefix)
		}

		for _, pageNumber := range selectedPages {
			prefix := filepath.Join(tempDir, fmt.Sprintf("page-%06d", pageNumber))
			if err := runPDFToPPM(
				"-png", "-r", "150",
				"-f", strconv.Itoa(pageNumber),
				"-l", strconv.Itoa(pageNumber),
				"-singlefile",
				pdfPath,
				prefix,
			); err != nil {
				return fmt.Errorf("render page %d: %w", pageNumber, err)
			}
		}
		return nil
	}

	if err := renderWithPoppler(); err != nil {
		log.Printf("Poppler rendering unavailable (%v); using local PyMuPDF fallback", err)
		if err := renderPDFWithPython(pdfPath, tempDir, selectedPages); err != nil {
			return nil, err
		}
	}

	return collectRenderedPages(tempDir)
}

func collectRenderedPages(tempDir string) ([]renderedPage, error) {
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, err
	}

	pages := make([]renderedPage, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(name), ".png") {
			continue
		}

		stem := strings.TrimSuffix(name, filepath.Ext(name))
		numberText := stem[strings.LastIndex(stem, "-")+1:]
		pageNumber, err := strconv.Atoi(numberText)
		if err != nil {
			continue
		}
		pages = append(pages, renderedPage{
			PageNumber: pageNumber,
			Path:       filepath.Join(tempDir, name),
		})
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].PageNumber < pages[j].PageNumber
	})
	if len(pages) == 0 {
		return nil, fmt.Errorf("no PDF pages were rendered")
	}
	return pages, nil
}

func recognizeRenderedPages(pages []renderedPage, lightMode bool) (*engine.OcrResult, error) {
	result := &engine.OcrResult{Pages: make([]engine.OcrPage, 0, len(pages))}

	for index, rendered := range pages {
		log.Printf("OCR page %d (%d/%d)", rendered.PageNumber, index+1, len(pages))
		pageBytes, err := os.ReadFile(rendered.Path)
		if err != nil {
			return nil, fmt.Errorf("read rendered page %d: %w", rendered.PageNumber, err)
		}

		imageValue, _, err := image.Decode(bytes.NewReader(pageBytes))
		if err != nil {
			return nil, fmt.Errorf("decode rendered page %d: %w", rendered.PageNumber, err)
		}

		bounds := imageValue.Bounds()
		input := pageBytes
		width, height := bounds.Dx(), bounds.Dy()
		if ActiveEngine.InputKind() == engine.InputBGRA {
			input, width, height = prepareImage(imageValue)
		}

		pageResult, err := performOCR(input, width, height, lightMode)
		if err != nil {
			return nil, fmt.Errorf("OCR page %d: %w", rendered.PageNumber, err)
		}
		if len(pageResult.Pages) == 0 {
			continue
		}

		page := pageResult.Pages[0]
		page.PageNumber = rendered.PageNumber
		result.Pages = append(result.Pages, page)
	}

	return result, nil
}

func executableDirectory() string {
	executable, err := os.Executable()
	if err == nil {
		return filepath.Dir(executable)
	}
	workingDirectory, _ := os.Getwd()
	return workingDirectory
}

func findPythonRuntime() (string, error) {
	baseDir := executableDirectory()
	candidates := []string{
		filepath.Join(baseDir, ".venv", "Scripts", "python.exe"),
		filepath.Join(baseDir, ".venv", "bin", "python"),
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(workingDirectory, ".venv", "Scripts", "python.exe"),
			filepath.Join(workingDirectory, ".venv", "bin", "python"),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if pythonPath, err := exec.LookPath("python"); err == nil {
		return pythonPath, nil
	}
	return "", fmt.Errorf("Python with PyMuPDF was not found")
}

// helperSubdirs are searched under each base directory. The Python helpers
// live in fallback/ in the source tree, but deployments usually flatten
// everything next to the executable, so both layouts must work.
var helperSubdirs = []string{"", "fallback"}

func findHelper(name string) (string, error) {
	var bases []string
	if dir := executableDirectory(); dir != "" {
		bases = append(bases, dir)
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		bases = append(bases, workingDirectory)
	}

	var candidates []string
	for _, base := range bases {
		for _, sub := range helperSubdirs {
			candidates = append(candidates, filepath.Join(base, sub, name))
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s was not found (looked in %v)", name, candidates)
}

func renderPDFWithPython(inputPath, outputDir string, selectedPages []int) error {
	pythonPath, err := findPythonRuntime()
	if err != nil {
		return err
	}
	helperPath, err := findHelper("pdf_render.py")
	if err != nil {
		return err
	}

	pageValues := make([]string, len(selectedPages))
	for index, pageNumber := range selectedPages {
		pageValues[index] = strconv.Itoa(pageNumber)
	}
	command := exec.Command(
		pythonPath,
		helperPath,
		inputPath,
		outputDir,
		strings.Join(pageValues, ","),
		strconv.Itoa(MaxImageDimension),
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PyMuPDF rendering failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func createSearchablePDF(inputPath, outputPath, tempDir string, result *engine.OcrResult) error {
	payloadPath := filepath.Join(tempDir, "ocr-result.json")
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := os.WriteFile(payloadPath, payload, 0600); err != nil {
		return err
	}

	pythonPath, err := findPythonRuntime()
	if err != nil {
		return err
	}
	helperPath, err := findHelper("pdf_overlay.py")
	if err != nil {
		return err
	}
	command := exec.Command(pythonPath, helperPath, inputPath, outputPath, payloadPath)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("searchable PDF overlay failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func writePDFDownload(w http.ResponseWriter, filename string, contents []byte) {
	filename = strings.ReplaceAll(filepath.Base(filename), `"`, "")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(contents)))
	_, _ = w.Write(contents)
}

func HandleOcrPdf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if ActiveEngine == nil {
		writeJSONError(w, http.StatusInternalServerError, "OCR Engine is not initialized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Missing file parameter")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to read PDF: "+err.Error())
		return
	}

	selectedPages, err := parsePageRange(r.FormValue("pages"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	makeSearchable, _ := strconv.ParseBool(r.FormValue("make_searchable"))
	lightMode, _ := strconv.ParseBool(r.FormValue("light_mode"))
	started := time.Now()

	textPages, _, textErr := readPDFTextPages(fileBytes, toPageSet(selectedPages))
	if textErr == nil {
		nativeResult, nativeText, allPagesHaveText := nativeTextResult(textPages)
		if allPagesHaveText {
			duration := time.Since(started).Seconds()
			entryType := "pdf-text-layer"
			if makeSearchable {
				entryType = "pdf-searchable-existing"
			}
			AddHistoryEntry(header.Filename, entryType, len(fileBytes), duration, len(nativeResult.Pages))

			if makeSearchable {
				name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + "_searchable.pdf"
				writePDFDownload(w, name, fileBytes)
				return
			}

			response := OcrResponse{
				Filename:        header.Filename,
				DurationSeconds: duration,
				Result:          nativeResult,
				Text:            nativeText,
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(response)
			return
		}
	}

	tempDir, err := os.MkdirTemp("", "tank-ocr-pdf-*")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create temporary directory: "+err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "input.pdf")
	if err := os.WriteFile(inputPath, fileBytes, 0600); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to prepare PDF: "+err.Error())
		return
	}

	renderedPages, err := renderPDFPages(inputPath, tempDir, selectedPages)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := recognizeRenderedPages(renderedPages, lightMode)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	duration := time.Since(started).Seconds()

	if makeSearchable {
		outputPath := filepath.Join(tempDir, "searchable.pdf")
		if err := createSearchablePDF(inputPath, outputPath, tempDir, result); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		outputBytes, err := os.ReadFile(outputPath)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to read searchable PDF: "+err.Error())
			return
		}
		AddHistoryEntry(header.Filename, "pdf-searchable", len(fileBytes), duration, len(result.Pages))
		name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + "_searchable.pdf"
		writePDFDownload(w, name, outputBytes)
		return
	}

	AddHistoryEntry(header.Filename, "pdf-ocr", len(fileBytes), duration, len(result.Pages))
	response := OcrResponse{
		Filename:        header.Filename,
		DurationSeconds: duration,
		Result:          result,
		Text:            flattenText(result),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}
