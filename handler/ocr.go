package handler

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"tank-ocr/engine"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var (
	// ActiveEngine is set in main.go
	ActiveEngine engine.OcrEngine
	// OcrMutex serialises all PerformOcr calls for Screen AI runtime safety.
	OcrMutex sync.Mutex
)

type OcrResponse struct {
	Filename        string            `json:"filename"`
	DurationSeconds float64           `json:"duration_seconds"`
	Result          *engine.OcrResult `json:"result"`
	Text            string            `json:"text"`
}

func HandleOcr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if ActiveEngine == nil {
		writeJSONError(w, http.StatusInternalServerError, "OCR Engine is not initialized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Missing file parameter")
		return
	}
	defer file.Close()

	lightMode := r.FormValue("light_mode") == "true"

	// Read file bytes
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to read file: "+err.Error())
		return
	}

	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(fileBytes))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid image format: "+err.Error())
		return
	}
	const maxDecodedPixels = 100_000_000
	if imageConfig.Width <= 0 || imageConfig.Height <= 0 ||
		int64(imageConfig.Width)*int64(imageConfig.Height) > maxDecodedPixels {
		writeJSONError(w, http.StatusBadRequest, "Image dimensions are invalid or too large")
		return
	}

	img, format, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid image format: "+err.Error())
		return
	}

	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	var input []byte
	var finalW, finalH int
	if ActiveEngine.InputKind() == engine.InputBGRA {
		log.Printf("Preparing %s image for raw BGRA OCR", format)
		input, finalW, finalH = prepareImage(img)
	} else {
		input, finalW, finalH = fileBytes, imgW, imgH
	}

	t0 := time.Now()

	ocrResult, err := performOCR(input, finalW, finalH, lightMode)

	if err != nil {
		log.Printf("OCR failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "OCR execution failed: "+err.Error())
		return
	}

	duration := time.Since(t0).Seconds()
	plainText := flattenText(ocrResult)

	// Add to history logger
	AddHistoryEntry(header.Filename, "image", len(fileBytes), duration, 1)

	// Build response
	resp := OcrResponse{
		Filename:        header.Filename,
		DurationSeconds: duration,
		Result:          ocrResult,
		Text:            plainText,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
