package wavreader

import (
	"fmt"
	"os"
	"os/exec"
)

// AudioOutput creates a WAV Writer that pipes the audio stream to the local sound card
func AudioOutput() (Writer, error) {
	return defaultConfig.AudioOutput()
}

// AudioOutput creates a WAV Writer that pipes the audio stream to the local sound card
func (c Config) AudioOutput() (Writer, error) {
	format := c.playbackFormat()

	if format.Format != 1 {
		return nil, fmt.Errorf("unknown output format %d", format.Format)
	}

	rawAudio := fmt.Sprintf("channels=%d:rate=%d:samplesize=%d",
		format.Channels,
		format.Rate,
		(format.Bits+7)/8,
	)
	mpl := exec.Command(c.MPlayerPath,
		"-really-quiet",
		"-noconsolecontrols", "-nomouseinput", "-nolirc",
		"-cache", "1024",
		"-demuxer", "rawaudio",
		"-rawaudio", rawAudio,
		"-")

	// FIXME: Have this go somewhere more productive
	mpl.Stdout = os.Stdout
	mpl.Stderr = os.Stderr

	stdin, err := mpl.StdinPipe()
	if err != nil {
		return nil, err
	}

	mpl.Start()

	rv := &wavWriter{
		target:        stdin,
		targetProcess: mpl,
		initialized:   true,
		format:        format,
	}

	return rv, nil
}
