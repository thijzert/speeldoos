package wavreader

import (
	"fmt"
	"io"
)

var (
	errUninitialized error = fmt.Errorf("reader not yet initialized")
	errParse         error = fmt.Errorf("parse error")
)

// A Reader represents a read-only audio stream
type Reader interface {
	io.ReadCloser
	Formatter

	// Init parses the WAV header and populates the StreamFormat fields
	Init()

	// Size returns the expected size for this audio stream
	Size() int

	// SetSize sets the expected size to a known value
	SetSize(int)
}

type wavReader struct {
	source      io.ReadCloser
	errorState  error
	initialized bool
	size        int
	bytesRead   int
	format      StreamFormat
}

// New creates a Reader from a stream encoded in the WAV file format
func New(source io.ReadCloser) Reader {
	rv := &wavReader{source: source, initialized: false}
	return rv
}

// Pipe creates a synchronous in-memory pipe, with the specified audio format.
// It can be used to connect code expecting a Reader with code expecting a Writer.
func Pipe(format StreamFormat) (Reader, Writer) {
	pr, pw := io.Pipe()
	rv := &wavReader{
		source:      pr,
		initialized: true,
		format:      format,
	}
	rw := &wavWriter{
		target:      pw,
		initialized: true,
		format:      format,
	}

	return rv, rw
}

// Format returns the audio stream format for the reader
func (w *wavReader) Format() StreamFormat {
	return w.format
}

// Size returns the expected size for this audio stream
func (w *wavReader) Size() int {
	return w.size
}

// SetSize sets the expected size to a known value
func (w *wavReader) SetSize(s int) {
	w.size = s
}

// Init parses the WAV header and populates the StreamFormat fields
func (w *wavReader) Init() {
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
		w.errorState = errParse
		return
	}

	totalLength := atoi(b[4:8])

	if string(b[8:12]) != "WAVE" {
		w.errorState = errParse
		return
	}
	if string(b[12:15]) != "fmt" {
		w.errorState = errParse
		return
	}

	headerLength := atoi(b[16:20])
	if headerLength < 16 {
		w.errorState = errParse
		return
	}

	dataChunkStart := 36
	w.format.Format = atoi(b[20:22])
	if w.format.Format == 0xfffe {
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
			w.format.Format = 1
		}
	}
	if w.format.Format != 1 {
		w.errorState = errParse
		return
	}

	w.format.Channels = atoi(b[22:24])
	w.format.Rate = atoi(b[24:28])
	w.format.Bits = atoi(b[34:36])

	bytesPerSecond := atoi(b[28:32])
	expectedBytesPerSecond := w.format.BytesPerSample() * w.format.Rate
	if bytesPerSecond != expectedBytesPerSecond {
		w.errorState = errParse
		return
	}

	bytesPerSample := atoi(b[32:34])
	expectedBytesPerSample := w.format.BytesPerSample()
	if bytesPerSample != expectedBytesPerSample {
		w.errorState = errParse
		return
	}

	// Data Chunk
	dc := b[dataChunkStart:]

	if string(dc[0:4]) != "data" {
		w.errorState = errParse
		return
	}

	w.size = atoi(dc[4:8])

	if totalLength != w.size+headerLength+20 {
		w.errorState = errParse
	}
}

func (w *wavReader) Read(b []byte) (int, error) {
	if w.errorState != nil {
		return 0, w.errorState
	}
	if !w.initialized {
		return 0, errUninitialized
	}

	//if len(b) > (w.size-w.bytesRead) {
	//	b = b[:w.size-w.bytesRead]
	//}
	n, err := w.source.Read(b)

	w.bytesRead += n
	if err != nil {
		w.errorState = err
	}

	return n, err
}

// WriteTo writes data to w until there's no more data to write or when an error
// occurs. The return value n is the number of bytes written. Any error
// encountered during the write is also returned.
func (w *wavReader) WriteTo(dest io.Writer) (int64, error) {
	if wri, ok := dest.(Writer); ok {
		if wri.Format() == w.format {
			return io.Copy(wri, w.source)
		}

		written, err := doConversion(wri, w)
		if err == io.EOF {
			err = nil
		}
		return written, err
	}

	return io.Copy(dest, w.source)
}

// Close closes the reader and frees up any held resources
func (w *wavReader) Close() error {
	if w.errorState != nil {
		return w.errorState
	}

	w.errorState = w.source.Close()
	return w.errorState
}
