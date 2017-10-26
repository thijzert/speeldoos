package wavreader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Config struct {
	// Path to `lame` binary
	LamePath string

	// MP3 encoder settings: max bitrate (ABR mode)
	MaxBitrate int

	// MP3 encoder settings: VBR quality preset
	VBRQuality int

	// Path to `flac` binary
	FlacPath string
}

var defaultConfig Config

func (c Config) lame() string {
	if c.LamePath != "" {
		return c.LamePath
	}
	return "lame"
}

func (c Config) flac() string {
	if c.FlacPath != "" {
		return c.FlacPath
	}
	return "flac"
}

type mp3Writer struct {
	cmd    *exec.Cmd
	mp3Out io.Writer
	input  io.WriteCloser
	output io.ReadCloser
}

func ToMP3(mp3Out io.Writer, channels, sampleRate, bitsPerSample int) (io.WriteCloser, error) {
	return defaultConfig.ToMP3(mp3Out, channels, sampleRate, bitsPerSample)
}

func (c Config) ToMP3(mp3Out io.Writer, channels, sampleRate, bitsPerSample int) (io.WriteCloser, error) {
	var err error

	var mode string
	if channels == 1 {
		mode = "m"
	} else if channels == 2 {
		mode = "j" // joint stereo
	} else {
		return nil, fmt.Errorf("unsupported number of channels %d", channels)
	}

	mw := &mp3Writer{mp3Out: mp3Out}

	lamecmd := []string{
		"-r", "--quiet", "--replaygain-accurate", "--id3v2-only",
		"-s", fmt.Sprintf("%g", float64(sampleRate)/1000.0),
		"--bitwidth", fmt.Sprintf("%d", bitsPerSample),
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
