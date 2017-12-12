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

func Pipe() (*Reader, *io.PipeWriter) {
	pr, pw := io.Pipe()
	rv := &Reader{
		source:      pr,
		initialized: true,
	}

	return rv, pw
}

func (w *Reader) Init() {
	if w.initialized {
		return
	}
	w.initialized = true

	b := make([]byte, 44, 68)
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
	dataChunkStart := 36
	w.FormatType = atoi(b[20:22])
	if w.FormatType == 0xfffe {
		b = b[:68]
		_, err := io.ReadFull(w.source, b[44:])
		if err != nil {
			w.errorState = err
			return
		}
		extendedData := atoi(b[36:38])
		if extendedData > 22 {
			extraExtra := make([]byte, extendedData-22)
			_, err := io.ReadFull(w.source, extraExtra)
			if err != nil {
				w.errorState = err
				return
			}

			b = append(b, extraExtra...)
		}

		dataChunkStart += 2 + extendedData

		// Check the extended format GUID
		if string(b[44:60]) == "\x01\x00\x00\x00\x00\x00\x10\x00\x80\x00\x00\xaa\x00\x38\x9b\x71" {
			// Whew, still PCM
			w.FormatType = 1
		}
	}
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

	// Data Chunk
	dc := b[dataChunkStart:]

	if string(dc[0:4]) != "data" {
		log.Printf("Expected: \"data\" or \"fmt \"; got: \"%s\" (%02x)", dc[0:4], dc[0:4])
		w.errorState = parseError
		return
	}

	w.Size = atoi(dc[4:8])
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
