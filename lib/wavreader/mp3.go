package wavreader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type mp3Writer struct {
	cmd    *exec.Cmd
	mp3Out io.Writer
	input  io.WriteCloser
	output io.ReadCloser
}

func ToMP3(mp3Out io.Writer, format StreamFormat) (io.WriteCloser, error) {
	return defaultConfig.ToMP3(mp3Out, format)
}

func (c Config) ToMP3(mp3Out io.Writer, format StreamFormat) (io.WriteCloser, error) {
	var err error

	var mode string
	if format.Channels == 1 {
		mode = "m"
	} else if format.Channels == 2 {
		mode = "j" // joint stereo
	} else {
		return nil, fmt.Errorf("unsupported number of channels %d", format.Channels)
	}

	mw := &mp3Writer{mp3Out: mp3Out}

	lamecmd := []string{
		"-r", "--quiet", "--replaygain-accurate", "--id3v2-only",
		"-s", fmt.Sprintf("%g", float64(format.Rate)/1000.0),
		"--bitwidth", fmt.Sprintf("%d", format.Bits),
		"-m", mode,
	}
	if c.MaxBitrate > 0 {
		lamecmd = append(lamecmd, "-B", fmt.Sprintf("%d", c.MaxBitrate))
	} else {
		lamecmd = append(lamecmd, "--vbr-new", fmt.Sprintf("-V%d", c.VBRQuality))
	}
	lamecmd = append(lamecmd, "-", "-")

	mw.cmd = exec.Command(c.lame(), lamecmd...)

	mw.cmd.Stderr = os.Stderr

	mw.output, err = mw.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	mw.input, err = mw.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	err = mw.cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		io.Copy(mp3Out, mw.output)
		mw.cmd.Wait()
		mw.output.Close()
		mw.input.Close()
	}()

	return mw, nil
}

func (mw *mp3Writer) Write(buf []byte) (int, error) {
	if mw.cmd.ProcessState != nil && mw.cmd.ProcessState.Exited() && !mw.cmd.ProcessState.Success() {
		return 0, fmt.Errorf("error writing MP3 file")
	}
	n, err := writeAll(mw.input, buf)
	return n, err
}

func (mw *mp3Writer) Close() error {
	mw.input.Close()
	err := mw.cmd.Wait()
	if err != nil {
		return err
	}
	if mw.cmd.ProcessState != nil && mw.cmd.ProcessState.Exited() && !mw.cmd.ProcessState.Success() {
		return fmt.Errorf("error writing MP3 file")
	}

	return nil
}
