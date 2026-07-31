package engine

type InputKind int

const (
	InputEncodedImage InputKind = iota
	InputBGRA
)

// BoundingBox represents word or line coordinates in pixels
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Angle  float64 `json:"angle,omitempty"`
}

// OcrWord represents a single recognized word
type OcrWord struct {
	Text        string       `json:"text"`
	Confidence  *float64     `json:"confidence,omitempty"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
}

// OcrLine represents a line of words
type OcrLine struct {
	Text        string       `json:"text"`
	Words       []OcrWord    `json:"words"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
}

// OcrBlock represents a paragraph or section of text lines
type OcrBlock struct {
	BlockType   string       `json:"block_type"` // e.g. "paragraph"
	Lines       []OcrLine    `json:"lines"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
}

// OcrPage represents a single document page
type OcrPage struct {
	PageNumber int        `json:"page_number"`
	Blocks     []OcrBlock `json:"blocks"`
	Width      float64    `json:"width,omitempty"`
	Height     float64    `json:"height,omitempty"`
}

// OcrResult represents the structured hierarchical OCR result
type OcrResult struct {
	Pages []OcrPage `json:"pages"`
}

// OcrEngine defines the pluggable interface for executing OCR
type OcrEngine interface {
	Init() error
	PerformOcr(imageBytes []byte, width, height int, lightMode bool) (*OcrResult, error)
	InputKind() InputKind
	Version() (string, error)
	Name() string
	Close() error
}
