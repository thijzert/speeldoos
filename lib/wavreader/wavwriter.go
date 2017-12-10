package wavreader

import (
	"io"
	"os"
)

type Writer struct {
	target        io.WriteCloser
	fixedSize     int
	initialized   bool
	observedSize  int
	FormatType    int
	Channels      int
	SampleRate    int
	BitsPerSample int
}

func NewWriter(target io.WriteCloser, formatType, channels, sampleRate, bitsPerSample int) *Writer {
	rv := &Writer{
		target:        target,
		initialized:   false,
		FormatType:    formatType,
		Channels:      channels,
		SampleRate:    sampleRate,
		BitsPerSample: bitsPerSample,
	}

	return rv
}

func (w *Writer) Init(fixedSize int) error {
	b := make([]byte, 44)

	stoa(b[0:4], "RIFF")
	itoa(b[4:8], fixedSize+36)
	stoa(b[8:12], "WAVE")
	stoa(b[12:16], "fmt ")
	itoa(b[16:20], 16)
	itoa(b[20:22], w.FormatType)
	itoa(b[22:24], w.Channels)
	itoa(b[24:28], w.SampleRate)
	itoa(b[28:32], (w.SampleRate*w.BitsPerSample*w.Channels)/8)
	itoa(b[32:34], (w.BitsPerSample*w.Channels)/8)
	itoa(b[34:36], w.BitsPerSample)
	stoa(b[36:40], "data")
	itoa(b[40:44], fixedSize)

	_, err := writeAll(w.target, b)
	if err != nil {
		return err
	}

	w.initialized = true

	return nil
}

func itoa(a []byte, i int) {
	for j, _ := range a {
		a[j] = byte(i & 0xff)
		i >>= 8
	}
}
func stoa(a []byte, s string) {
	b := []byte(s)
	for i, c := range b {
		a[i] = c
	}
}

func writeAll(wr io.Writer, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		i, err := wr.Write(buf[n:])
		n += i
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (w *Writer) Write(buf []byte) (int, error) {
	if !w.initialized {
		w.Init(0xffffffd3)
	}
	n, err := writeAll(w.target, buf)
	w.observedSize += n

	return n, err
}

func (w *Writer) Close() error {
	if !w.initialized {
		w.Init(0xffffffd3)
	}

	if f, ok := w.target.(*os.File); ok {
		_, err := f.Seek(0, 0)
		if err == nil {
			w.Init(w.observedSize)
		}
	}

	rv := w.target.Close()
	return rv
}
