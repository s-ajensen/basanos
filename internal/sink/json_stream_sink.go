package sink

import (
	"encoding/json"
	"fmt"
	"io"
)

type JsonStreamSink struct {
	writer io.Writer
}

func NewJsonStreamSink(w io.Writer) Sink {
	return &JsonStreamSink{writer: w}
}

func (s *JsonStreamSink) Emit(e any) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}
