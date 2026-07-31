package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type LocalFallbackEngine struct {
	url    string
	client *http.Client
}

func NewLocalFallbackEngine(url string) *LocalFallbackEngine {
	return &LocalFallbackEngine{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (l *LocalFallbackEngine) Init() error {
	if l.url == "" {
		return errors.New("local fallback URL is empty. Please configure it in config.json")
	}
	return nil
}

type FallbackResponse struct {
	Text   string         `json:"text"`
	Result []FallbackItem `json:"result"`
}

type FallbackItem struct {
	Text       string      `json:"text"`
	Confidence float64     `json:"confidence"`
	Box        [][]float64 `json:"box"` // [[x0, y0], [x1, y1], [x2, y2], [x3, y3]]
}

func (l *LocalFallbackEngine) PerformOcr(imageBytes []byte, width, height int, lightMode bool) (*OcrResult, error) {
	// Prepare multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, bytes.NewReader(imageBytes)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", l.url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("local fallback OCR request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("local fallback OCR returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var fallbackResp FallbackResponse
	if err := json.Unmarshal(respBytes, &fallbackResp); err != nil {
		// Attempt to parse simple string response as fallback
		return &OcrResult{
			Pages: []OcrPage{
				{
					PageNumber: 1,
					Width:      float64(width),
					Height:     float64(height),
					Blocks: []OcrBlock{
						{
							BlockType: "paragraph",
							Lines: []OcrLine{
								{
									Text: string(respBytes),
								},
							},
						},
					},
				},
			},
		}, nil
	}

	// Map fallback items to OcrResult
	var lines []OcrLine
	for _, item := range fallbackResp.Result {
		var box *BoundingBox
		if len(item.Box) >= 4 && len(item.Box[0]) >= 2 && len(item.Box[2]) >= 2 {
			x0, y0 := item.Box[0][0], item.Box[0][1]
			x2, y2 := item.Box[2][0], item.Box[2][1]
			box = &BoundingBox{
				X:      x0,
				Y:      y0,
				Width:  x2 - x0,
				Height: y2 - y0,
			}
		}

		conf := item.Confidence
		lines = append(lines, OcrLine{
			Text: item.Text,
			Words: []OcrWord{
				{
					Text:        item.Text,
					Confidence:  &conf,
					BoundingBox: box,
				},
			},
			BoundingBox: box,
		})
	}

	return &OcrResult{
		Pages: []OcrPage{
			{
				PageNumber: 1,
				Blocks: []OcrBlock{
					{
						BlockType: "paragraph",
						Lines:     lines,
					},
				},
				Width:  float64(width),
				Height: float64(height),
			},
		},
	}, nil
}

func (l *LocalFallbackEngine) Version() (string, error) {
	return "Local Fallback Engine", nil
}

func (l *LocalFallbackEngine) InputKind() InputKind {
	return InputEncodedImage
}

func (l *LocalFallbackEngine) Name() string {
	return "Local Fallback (PaddleOCR)"
}

func (l *LocalFallbackEngine) Close() error {
	return nil
}
