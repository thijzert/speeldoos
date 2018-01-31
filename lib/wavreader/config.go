package wavreader

type StreamFormat struct {
	Format   int
	Channels int
	Rate     int
	Bits     int
}

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
