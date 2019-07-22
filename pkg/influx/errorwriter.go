package influx

import (
	"bytes"
)

// errorWriter is a helper class that writes to a buffer until an error occurs.
// Any error encountered will be stored until retrieved with Result().
// Subsequent writes will be silent no-ops if an error has occurred.
type errorWriter struct {
	w   *bytes.Buffer
	err error
}

func (ew *errorWriter) WriteString(data string) {
	if ew.err != nil {
		return
	}
	_, err := ew.w.WriteString(data)
	if err != nil {
		ew.err = err
	}
}

func (ew *errorWriter) Result() ([]byte, error) {
	if ew.err != nil {
		return nil, ew.err
	}
	return ew.w.Bytes(), nil
}
