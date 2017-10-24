package wavreader

import (
	"fmt"
	"io"
	"os/exec"
)

type flacReader struct {
	cmd    *exec.Cmd
	flacIn io.ReadCloser
	input  io.WriteCloser
	output io.ReadCloser
}

func newFlacReader(flacIn io.ReadCloser) (*flacReader, error) {
	var err error

	fr := &flacReader{flacIn: flacIn}
	// FIXME: make the path to flac binary configurable
	fr.cmd = exec.Command("flac", "-s", "-c", "-d", "-")

	fr.output, fr.cmd.Stdout = io.Pipe()
	fr.input, err = fr.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	err = fr.cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		io.Copy(fr.input, flacIn)
		fr.flacIn.Close()
		fr.input.Close()
		fr.cmd.Wait()
		fr.output.Close()
	}()

	return fr, nil
}

func (fr *flacReader) Read(buf []byte) (int, error) {
	if fr.cmd.ProcessState != nil && fr.cmd.ProcessState.Exited() && !fr.cmd.ProcessState.Success() {
		return 0, fmt.Errorf("error decoding FLAC file")
	}
	n, err := fr.output.Read(buf)
	return n, err
}

func (fr *flacReader) Close() error {
	fr.output.Close()
	fr.flacIn.Close()
	fr.input.Close()

	return fr.cmd.Wait()
}

func FromFLAC(in io.ReadCloser) (*Reader, error) {
	wavout, err := newFlacReader(in)
	if err != nil {
		return nil, err
	}
	return New(wavout), nil
}
