package wavreader

import (
	"fmt"
	"io"
	"log"
)

var (
	uninitializedError error = fmt.Errorf("reader not yet initialized")
	parseError         error = fmt.Errorf("parse error")
)

type Reader struct {
	source        io.ReadCloser
	errorState    error
	initialized   bool
	Size          int
	bytesRead     int
	FormatType    int
	Channels      int
	SampleRate    int
	BitsPerSample int
}

func New(source io.ReadCloser) *Reader {
	rv := &Reader{source: source, initialized: false}
	return rv
}

func (w *Reader) Init() {
	if w.initialized {
		return
	}
	w.initialized = true

	b := make([]byte, 44)
	_, err := io.ReadFull(w.source, b)
	if err != nil {
		w.errorState = err
		return
	}

	if string(b[0:4]) != "RIFF" {
		log.Printf("Expected: \"RIFF\"; got: \"%s\" (%02x)", b[0:4], b[0:4])
		w.errorState = parseError
		return
	}

	//log.Printf("total file size: %d bytes", atoi(b[4:8])+8)

	if string(b[8:12]) != "WAVE" {
		log.Printf("Expected: \"WAVE\"; got: \"%s\" (%02x)", b[8:12], b[8:12])
		w.errorState = parseError
		return
	}
	if string(b[12:15]) != "fmt" {
		log.Printf("Expected: \"fmt\\0\" or \"fmt \"; got: \"%s\" (%02x)", b[12:16], b[12:16])
		w.errorState = parseError
		return
	}

	//log.Printf("Format header length: %d bytes", atoi(b[16:20]))
	w.FormatType = atoi(b[20:22])
	if w.FormatType != 1 {
		log.Printf("Expected format PCM (1); got unknown format ID %d", w.FormatType)
		w.errorState = parseError
		return
	}

	w.Channels = atoi(b[22:24])
	w.SampleRate = atoi(b[24:28])
	w.BitsPerSample = atoi(b[34:36])

	bytesPerSecond := atoi(b[28:32])
	if bytesPerSecond*8 != (w.BitsPerSample * w.SampleRate * w.Channels) {
		log.Printf("Invalid number of bytes per second: got %d; expected %d (=%d*%d*%d/8)", bytesPerSecond, (w.BitsPerSample*w.SampleRate*w.Channels)/8, w.BitsPerSample, w.SampleRate, w.Channels)
		w.errorState = parseError
		return
	}

	bytesPerSample := atoi(b[32:34])
	if bytesPerSample*8 != (w.BitsPerSample * w.Channels) {
		log.Printf("Invalid number of bytes per sample: got %d; expected %d (=%d*%d/8)", bytesPerSample, (w.BitsPerSample*w.Channels)/8, w.BitsPerSample, w.Channels)
		w.errorState = parseError
		return
	}

	if string(b[36:40]) != "data" {
		log.Printf("Expected: \"data\" or \"fmt \"; got: \"%s\" (%02x)", b[36:40], b[36:40])
		w.errorState = parseError
		return
	}

	w.Size = atoi(b[40:44])
}

func atoi(buf []byte) int {
	var rv int = 0
	for i, b := range buf {
		rv |= int(b) << uint(i*8)
	}
	return rv
}

func (w *Reader) Read(b []byte) (int, error) {
	if w.errorState != nil {
		return 0, w.errorState
	}
	if !w.initialized {
		return 0, uninitializedError
	}

	//if len(b) > (w.Size-w.bytesRead) {
	//	b = b[:w.Size-w.bytesRead]
	//}
	n, err := w.source.Read(b)

	w.bytesRead += n
	if err != nil {
		w.errorState = err
	}

	return n, err
}

func (w *Reader) Close() error {
	if w.errorState != nil {
		return w.errorState
	}

	w.errorState = w.source.Close()
	return w.errorState
}
