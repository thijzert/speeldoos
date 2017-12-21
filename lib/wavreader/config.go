package wavreader

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
