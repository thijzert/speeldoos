package wavreader

import (
	"bytes"
	"fmt"
	"testing"
)

func TestWavHeader(t *testing.T) {
	w, err := parseWavString(t, "raff........................................xxxxxxxx")
	if err == nil {
	}
}

func parseWavString(t *testing.T, wav string) (*Reader, error) {
	reader := bytes.NewBufferString(wav)
	return New(reader), fmt.Errorf("not implemented")
}
