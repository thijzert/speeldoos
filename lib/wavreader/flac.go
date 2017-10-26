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

func (c Config) newFlacReader(flacIn io.ReadCloser) (*flacReader, error) {
	var err error

	fr := &flacReader{flacIn: flacIn}
	fr.cmd = exec.Command(c.flac(), "-s", "-c", "-d", "-")

	fr.output, err = fr.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
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
	if err != nil {
		// Treat successful exits as EOF
		if fr.cmd.ProcessState != nil && fr.cmd.ProcessState.Exited() && fr.cmd.ProcessState.Success() {
			return 0, io.EOF
		}
	}
	return n, err
}

func (fr *flacReader) Close() error {
	fr.output.Close()
	fr.flacIn.Close()
	fr.input.Close()

	return fr.cmd.Wait()
}

func FromFLAC(in io.ReadCloser) (*Reader, error) {
	return defaultConfig.FromFLAC(in)
}

func (c Config) FromFLAC(in io.ReadCloser) (*Reader, error) {
	wavout, err := c.newFlacReader(in)
	if err != nil {
		return nil, err
	}
	return New(wavout), nil
}
