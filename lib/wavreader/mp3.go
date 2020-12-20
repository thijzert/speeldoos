package wavreader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// ToMP3 creates a WAV Writer that encodes the output into an MP3 stream
func ToMP3(mp3Out io.Writer, format StreamFormat) (Writer, error) {
	return defaultConfig.ToMP3(mp3Out, format)
}

// ToMP3 creates a WAV Writer that encodes the output into an MP3 stream
func (c Config) ToMP3(mp3Out io.Writer, format StreamFormat) (Writer, error) {
	var err error

	var mode string
	if format.Channels == 1 {
		mode = "m"
	} else if format.Channels == 2 {
		mode = "j" // joint stereo
	} else {
		return nil, fmt.Errorf("unsupported number of channels %d", format.Channels)
	}

	lamecmd := []string{
		"-r", "--quiet", "--replaygain-accurate", "--id3v2-only",
		"-s", fmt.Sprintf("%g", float64(format.Rate)/1000.0),
		"--bitwidth", fmt.Sprintf("%d", format.Bits),
		"-m", mode,
	}
	if c.MaxBitrate > 0 {
		lamecmd = append(lamecmd, "--abr", fmt.Sprintf("%d", c.MaxBitrate))
	} else {
		lamecmd = append(lamecmd, "--vbr-new", fmt.Sprintf("-V%d", c.VBRQuality))
	}
	lamecmd = append(lamecmd, "-", "-")

	mw := &wavWriter{
		initialized: true,
		format:      format,
	}
	mw.targetProcess = exec.Command(c.lame(), lamecmd...)

	mw.targetProcess.Stderr = os.Stderr
	mw.targetProcess.Stdout = mp3Out

	mw.target, err = mw.targetProcess.StdinPipe()
	if err != nil {
		return nil, err
	}

	err = mw.targetProcess.Start()
	if err != nil {
		return nil, err
	}

	return mw, nil
}
