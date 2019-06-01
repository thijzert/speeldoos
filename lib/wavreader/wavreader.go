package wavreader

import (
	"fmt"
	"io"
)

var (
	uninitializedError error = fmt.Errorf("reader not yet initialized")
	parseError         error = fmt.Errorf("parse error")
)

type Reader interface {
	io.ReadCloser
	Formater

	Init()
	Size() int
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

func New(source io.ReadCloser) Reader {
	rv := &wavReader{source: source, initialized: false}
	return rv
}

func Pipe(format StreamFormat) (Reader, *Writer) {
	pr, pw := io.Pipe()
	rv := &wavReader{
		source:      pr,
		initialized: true,
		format:      format,
	}
	rw := &Writer{
		target:      pw,
		initialized: true,
		Format:      format,
	}

	return rv, rw
}

func (w *wavReader) Format() StreamFormat {
	return w.format
}

func (w *wavReader) Size() int {
	return w.size
}
func (w *wavReader) SetSize(s int) {
	w.size = s
}

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
		w.errorState = parseError
		return
	}

	totalLength := atoi(b[4:8])

	if string(b[8:12]) != "WAVE" {
		w.errorState = parseError
		return
	}
	if string(b[12:15]) != "fmt" {
		w.errorState = parseError
		return
	}

	headerLength := atoi(b[16:20])
	if headerLength < 16 {
		w.errorState = parseError
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
		w.errorState = parseError
		return
	}

	w.format.Channels = atoi(b[22:24])
	w.format.Rate = atoi(b[24:28])
	w.format.Bits = atoi(b[34:36])

	bytesPerSecond := atoi(b[28:32])
	expectedBytesPerSecond := (w.format.Channels*w.format.Rate*w.format.Bits + 7) / 8
	if bytesPerSecond != expectedBytesPerSecond {
		w.errorState = parseError
		return
	}

	bytesPerSample := atoi(b[32:34])
	expectedBytesPerSample := (w.format.Channels*w.format.Bits + 7) / 8
	if bytesPerSample != expectedBytesPerSample {
		w.errorState = parseError
		return
	}

	// Data Chunk
	dc := b[dataChunkStart:]

	if string(dc[0:4]) != "data" {
		w.errorState = parseError
		return
	}

	w.size = atoi(dc[4:8])

	if totalLength != w.size+headerLength+20 {
		w.errorState = parseError
	}
}

func (w *wavReader) Read(b []byte) (int, error) {
	if w.errorState != nil {
		return 0, w.errorState
	}
	if !w.initialized {
		return 0, uninitializedError
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

func (r *wavReader) WriteTo(w io.Writer) (int64, error) {
	if wri, ok := w.(*Writer); ok {
		if wri.Format == r.format {
			return io.Copy(wri.target, r)
		} else {
			written, err := doConversion(wri, r)
			if err == io.EOF {
				err = nil
			}
			return written, err
		}
	} else {
		return io.Copy(w, r.source)
	}
}

func (w *wavReader) Close() error {
	if w.errorState != nil {
		return w.errorState
	}

	w.errorState = w.source.Close()
	return w.errorState
}
