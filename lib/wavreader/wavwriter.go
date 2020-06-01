package wavreader

import (
	"io"
	"os"
	"os/exec"
)

type Writer interface {
	io.WriteCloser
	Formater

	Init(int) error
	CloseWithError(error) error
}

type wavWriter struct {
	target        io.WriteCloser
	targetProcess *exec.Cmd
	fixedSize     int
	initialized   bool
	observedSize  int
	format        StreamFormat
	errorState    error
}

func NewWriter(target io.WriteCloser, format StreamFormat) Writer {
	rv := &wavWriter{
		target:      target,
		initialized: false,
		format:      format,
	}

	return rv
}

func (w *wavWriter) Init(fixedSize int) error {
	b := make([]byte, 44)

	stoa(b[0:4], "RIFF")
	itoa(b[4:8], fixedSize+36)
	stoa(b[8:12], "WAVE")
	stoa(b[12:16], "fmt ")
	itoa(b[16:20], 16)
	itoa(b[20:22], w.Format().Format)
	itoa(b[22:24], w.Format().Channels)
	itoa(b[24:28], w.Format().Rate)
	itoa(b[28:32], (w.Format().Channels*w.Format().Rate*w.Format().Bits+7)/8)
	itoa(b[32:34], (w.Format().Channels*w.Format().Bits+7)/8)
	itoa(b[34:36], w.Format().Bits)
	stoa(b[36:40], "data")
	itoa(b[40:44], fixedSize)

	_, err := writeAll(w.target, b)
	if err != nil {
		return err
	}

	w.initialized = true

	return nil
}

func (w *wavWriter) Format() StreamFormat {
	return w.format
}

func (w *wavWriter) Size() int {
	return w.fixedSize
}
func (w *wavWriter) SetSize(s int) {
	w.fixedSize = s
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

func (w *wavWriter) Write(buf []byte) (int, error) {
	if w.errorState != nil {
		return 0, w.errorState
	}
	if !w.initialized {
		w.Init(0x7fffffd3)
	}
	n, err := writeAll(w.target, buf)
	w.observedSize += n

	return n, err
}

func (w *wavWriter) Close() error {
	return w.CloseWithError(io.EOF)
}

func (w *wavWriter) CloseWithError(er error) error {
	if !w.initialized {
		w.Init(w.observedSize)
	}

	w.errorState = er

	if f, ok := w.target.(*os.File); ok {
		_, err := f.Seek(0, 0)
		if err == nil {
			w.Init(w.observedSize)
		}
	}

	var rv error
	if pipe, ok := w.target.(*io.PipeWriter); ok {
		rv = pipe.CloseWithError(er)
	} else {
		rv = w.target.Close()
	}

	if w.targetProcess != nil {
		if rv == nil {
			rv = w.targetProcess.Wait()
		} else {
			w.targetProcess.Wait()
		}
	}

	if er != nil {
		return er
	} else if rv != nil {
		w.errorState = rv
	}
	return rv
}
