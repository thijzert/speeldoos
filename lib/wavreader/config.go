package wavreader

import "fmt"

// A StreamFormat wraps all options that define a PCM audio stream format
type StreamFormat struct {
	Format   int
	Channels int
	Rate     int
	Bits     int
}

func (s StreamFormat) String() string {
	if s.Format != 1 {
		return fmt.Sprintf("PCM %dch %dHZ %dbit", s.Channels, s.Rate, s.Bits)
	}

	if s.Rate%1000 != 0 {
		fr := float64(s.Rate) * 0.001
		return fmt.Sprintf("PCM %dch %gkHZ %dbit", s.Channels, fr, s.Bits)
	}

	return fmt.Sprintf("PCM %dch %dkHZ %dbit", s.Channels, s.Rate/1000, s.Bits)
}

// BytesPerSample returns the byte length of each complete sample
func (f StreamFormat) BytesPerSample() int {
	return (f.Channels*f.Bits + 7) / 8
}

var (
	// CD is the format used on audio CD's
	CD StreamFormat = StreamFormat{
		Format:   1,
		Channels: 2,
		Rate:     44100,
		Bits:     16,
	}
	// DAT is a common format used on digital media, e.g. DAT tapes
	DAT StreamFormat = StreamFormat{
		Format:   1,
		Channels: 2,
		Rate:     48000,
		Bits:     16,
	}
	// DOG contains a stream format that's really only discernible from the other two by dogs
	DOG StreamFormat = StreamFormat{
		Format:   1,
		Channels: 2,
		Rate:     192000,
		Bits:     24,
	}
)

// A Formatter interface designates which values have a stream format
type Formatter interface {
	Format() StreamFormat
}

// A Config wraps options common to most WAV operations
type Config struct {
	// Path to `lame` binary
	LamePath string

	// MP3 encoder settings: max bitrate (ABR mode)
	MaxBitrate int

	// MP3 encoder settings: VBR quality preset
	VBRQuality int

	// Path to `flac` binary
	FlacPath string

	// Path to `mplayer` binary
	MPlayerPath string

	// Stream format for audio output
	PlaybackFormat StreamFormat
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

func (c Config) mplayer() string {
	if c.MPlayerPath != "" {
		return c.MPlayerPath
	}
	return "mplayer"
}

func (c Config) playbackFormat() StreamFormat {
	if c.PlaybackFormat.Format != 0 {
		return c.PlaybackFormat
	}
	return StreamFormat{
		Format:   1,
		Channels: 2,
		Rate:     44100,
		Bits:     16,
	}
}
