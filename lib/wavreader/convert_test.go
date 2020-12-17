package wavreader

import (
	"io"
	"testing"
)

func TestConstant(t *testing.T) {
	samplesIn := make([]byte, 40)
	// Create a 125Hz sawtooth wave
	for i, _ := range samplesIn {
		samplesIn[i] = byte(36 * (i % 8))
	}

	rIn, wIn := io.Pipe()
	rOut, wOut := io.Pipe()

	wavIn := &wavReader{
		source:      rIn,
		initialized: true,
		format:      StreamFormat{Format: 1, Channels: 1, Rate: 1000, Bits: 8},
		size:        len(samplesIn),
	}

	wavOut := &wavWriter{
		target:      wOut,
		initialized: true,
		format:      StreamFormat{Format: 1, Channels: 1, Rate: 8000, Bits: 8},
	}

	go func() {
		n := 0
		for n < len(samplesIn) {
			i, err := wIn.Write(samplesIn[n:])
			n += i
			if err != nil {
				t.Errorf("pipe: %s", err)
				return
			}
		}
		wIn.Close()
	}()

	go func() {
		_, err := io.Copy(wavOut, wavIn)
		wavOut.Close()
		if err != nil {
			t.Errorf("convert: %s", err)
			return
		}
	}()

	samplesOut := make([]byte, 8*len(samplesIn))
	n, err := io.ReadFull(rOut, samplesOut)
	if err != nil {
		t.Errorf("readfull: only got %d bytes (wanted %d) %s", n, len(samplesOut), err)
	}

	t.Logf("In: %d", samplesIn)
	t.Logf("Out: %d", samplesOut)
	t.Logf("I wouldn't know how to test the output of the sample rate conversion. Root mean square error perhaps?")
}
