package output

import (
	"encoding/json"
	"io"
)

type Writer struct{}

func NewWriter() *Writer { return &Writer{} }

func (w *Writer) JSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
