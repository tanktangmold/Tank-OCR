package engine

import "testing"

func TestParseVisualAnnotationHandlesTruncatedData(t *testing.T) {
	inputs := [][]byte{
		{0x12},
		{0x12, 0xff},
		{0x12, 0x08, 0x1a, 0xff},
		{0x12, 0x04, 0x1d, 0x00},
	}

	for _, input := range inputs {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("parseVisualAnnotation panicked for %v: %v", input, recovered)
				}
			}()
			_ = parseVisualAnnotation(input)
		}()
	}
}
