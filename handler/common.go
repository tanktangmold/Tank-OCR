package handler

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"net/http"

	"tank-ocr/engine"

	xdraw "golang.org/x/image/draw"
)

var (
	MaxImageDimension = 2048
	HistoryLimit      = 50
)

func Configure(maxImageDimension, historyLimit int) {
	if maxImageDimension > 0 {
		MaxImageDimension = maxImageDimension
	}
	if historyLimit > 0 {
		HistoryLimit = historyLimit
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": message})
}

func prepareImage(img image.Image) ([]byte, int, int) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	maxEdge := width
	if height > maxEdge {
		maxEdge = height
	}

	if maxEdge > MaxImageDimension {
		scale := float64(MaxImageDimension) / float64(maxEdge)
		newWidth := max(1, int(float64(width)*scale))
		newHeight := max(1, int(float64(height)*scale))
		log.Printf("Resizing image from %dx%d to %dx%d", width, height, newWidth, newHeight)

		resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		xdraw.ApproxBiLinear.Scale(resized, resized.Bounds(), img, bounds, xdraw.Over, nil)
		img = resized
	}

	return imageToBGRA(img)
}

func imageToBGRA(img image.Image) ([]byte, int, int) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	bgra := make([]byte, width*height*4)

	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			bgra[index] = uint8(b >> 8)
			bgra[index+1] = uint8(g >> 8)
			bgra[index+2] = uint8(r >> 8)
			bgra[index+3] = uint8(a >> 8)
			index += 4
		}
	}

	return bgra, width, height
}

func performOCR(input []byte, width, height int, lightMode bool) (result *engine.OcrResult, err error) {
	OcrMutex.Lock()
	defer OcrMutex.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("OCR engine panic: %v", recovered)
		}
	}()

	return ActiveEngine.PerformOcr(input, width, height, lightMode)
}

func flattenText(result *engine.OcrResult) string {
	if result == nil {
		return ""
	}

	text := ""
	for pageIndex, page := range result.Pages {
		pageText := ""
		for _, block := range page.Blocks {
			blockText := ""
			for _, line := range block.Lines {
				if blockText != "" {
					blockText += "\n"
				}
				blockText += line.Text
			}
			if blockText != "" {
				if pageText != "" {
					pageText += "\n\n"
				}
				pageText += blockText
			}
		}
		if pageIndex > 0 {
			text += "\f"
		}
		text += pageText
	}
	return text
}
