package chunker

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
)

const mp3ReadAhead time.Duration = 30 * time.Second

var defaultMP3Config MP3ChunkConfig

func init() {
	defaultMP3Config.Audio.PlaybackFormat = wavreader.DAT
}

type MP3ChunkConfig struct {
	Context context.Context
	Audio   wavreader.Config
}

func (m MP3ChunkConfig) NewMP3() (Chunker, error) {
	r, w := io.Pipe()

	wavin, err := m.Audio.ToMP3(w, m.Audio.PlaybackFormat)
	if err != nil {
		return nil, err
	}

	rv := &mp3Chunker{
		audioIn: wavin,
		mp3out:  r,
		embargo: time.Now(),
		chcont: &chunkContainer{
			chunks: make([]chunk, 4000),
			start:  0,
			end:    0,
		},
	}

	go rv.splitChunks()

	return rv, nil
}

type mp3Chunker struct {
	audioIn wavreader.Writer
	mp3out  *io.PipeReader
	embargo time.Time
	chcont  *chunkContainer
}

func NewMP3() (Chunker, error) {
	return defaultMP3Config.NewMP3()
}

func (m *mp3Chunker) Init(fixedSize int) error {
	if m.chcont.errorState != nil {
		return m.chcont.errorState
	}
	return m.audioIn.Init(fixedSize)
}
func (m *mp3Chunker) Format() wavreader.StreamFormat {
	return m.audioIn.Format()
}
func (m *mp3Chunker) Write(buf []byte) (int, error) {
	if m.chcont.errorState != nil {
		return 0, m.chcont.errorState
	}
	return m.audioIn.Write(buf)
}
func (m *mp3Chunker) Close() error {
	if m.chcont.errorState != nil {
		return m.chcont.errorState
	}
	m.chcont.errorState = io.EOF
	return m.audioIn.Close()
}
func (m *mp3Chunker) CloseWithError(er error) error {
	if m.chcont.errorState != nil {
		return m.chcont.errorState
	}
	m.chcont.errorState = er
	if m.audioIn != nil {
		return m.audioIn.CloseWithError(er)
	} else {
		return er
	}
}

func (m *mp3Chunker) NewStream() (ChunkStream, error) {
	return m.chcont.NewStream()
}

func (m *mp3Chunker) splitChunks() {
	var hdr, nexthdr mp3header
	buf := make([]byte, 1024)
	firstOffset := 0
	offset := 0

	var err error
	var n, i, ct int
	for {
		if offset == len(buf) {
			m.CloseWithError(fmt.Errorf("buffer is full; no new header found"))
			return
		}
		n, err = m.mp3out.Read(buf[offset:])
		if err != nil {
			if err != io.EOF {
				// TODO: remove
				log.Printf("splitting hairs got me this error: %s", err)
			}
			break
		}

		if n == 0 || (n+offset) < 4 {
			continue
		}

		unread := buf[:n+offset]
		i, nexthdr = nextHeader(unread[firstOffset:], hdr)
		for i >= 0 {
			hdr = nexthdr
			chunk := unread[:firstOffset+i]
			m.chcont.AddChunk(chunk, m.embargo)

			m.embargo = m.embargo.Add(hdr.Duration())
			for time.Now().Add(mp3ReadAhead).Before(m.embargo) {
				time.Sleep(1 * time.Millisecond)
			}

			unread = unread[i+4:]
			i, nexthdr = nextHeader(unread[4:], hdr)
			firstOffset = 4
			ct++
		}

		copy(buf, unread)
		offset = len(unread)
	}
	m.CloseWithError(err)
}

type mp3header struct {
	A, B, C, D byte
}

var tabsel_123 [2][3][16]uint16 = [2][3][16]uint16{
	{
		{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
		{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
		{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
	},
	{
		{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	},
}

func (m mp3header) SyncWord() uint16 {
	return (uint16(m.A)<<4 | uint16(m.B)>>4) & 0xffe
}
func (m mp3header) Version() int {
	rv := m.B & 0x18 >> 3
	if rv&0x02 == 0x02 {
		// MPEG 1 or 2
		return int(rv&0x01 ^ 0x01)
	} else {
		// MPEG-2.5
		return 2
	}
}
func (m mp3header) Bitrate() uint16 {
	lsf := int((m.B >> 3 & 0x01) ^ 0x01)
	lay := 4 - int(m.B>>1&0x03)
	if lay == 0 || lay == 4 {
		return 0
	}
	bitrate_index := int(m.C >> 4)
	return tabsel_123[lsf][lay-1][bitrate_index]
}

var sampleTable [4][4]int32 = [4][4]int32{
	{44100, 48000, 32000, 22050},
	{16000, 11025, 12000, 8000},
	{22050, 24000, 16000, 11025},
	{1, 1, 1, 1},
}

func (m mp3header) SampleRate() int32 {
	samp := int(m.C&0x0c) >> 2
	ver := m.Version()
	return sampleTable[ver][samp]
}

var durationTable [4][4][4]time.Duration = [4][4][4]time.Duration{
	{
		// MPEG-1:  384, 1152, or 1152 samples per frame  (all durations in ns)
		{8707483, 8000000, 12000000, 17414966},
		{26122449, 24000000, 36000000, 52244898},
		{26122449, 24000000, 36000000, 52244898},
	},
	{
		// MPEG 2: -1,384,1152, or 576 samples per frame  (all durations in ns)
		{24000000, 34829932, 32000000, 48000000},
		{72000000, 104489796, 96000000, 144000000},
		{36000000, 52244898, 48000000, 72000000},
	},
	{
		// MPEG 2.5: -1,384,1152, or 576 samples per frame  (all durations in ns)
		{17414966, 16000000, 24000000, 34829932},
		{52244898, 48000000, 72000000, 104489796},
		{26122449, 24000000, 36000000, 52244898},
	},
}

func (m mp3header) Duration() time.Duration {
	if m.A != 0xff {
		return 0
	}

	ver := m.Version()
	lay := 4 - int(m.B>>1&0x03)
	samp := int(m.C&0x0c) >> 2

	return durationTable[ver][lay-1][samp]
}

func (m mp3header) Padding() int {
	return int(m.C >> 1 & 0x01)
}

// Framesize returns the total length of the MP3 frame, including its header
func (m mp3header) Framesize() int {
	ver := m.Version()
	lay := 4 - int(m.B>>1&0x03)
	if ver == 0 && lay == 3 {
		return (int(m.Bitrate()) * 144000 / int(m.SampleRate())) + m.Padding()
	}
	return 4
}

func (m mp3header) String() string {
	if m.SyncWord() != 0xffe {
		return "(not an MP3 header)"
	}

	ver := [4]string{"1.0", "2.0", "2.5", "x.x"}[m.Version()]
	lay := [4]string{"Unknown", "I", "II", "III"}[4-int(m.B>>1&0x03)]
	samp := m.SampleRate()
	br := m.Bitrate()

	return fmt.Sprintf("MPEG-%s layer %s; %dHz %dkbps", ver, lay, samp, br)
}

func nextHeader(buf []byte, last mp3header) (int, mp3header) {
	l := len(buf)
	if l < 4 {
		return -1, last
	}

	if last.A == 0xff {
		// The previous header was filled; see if we can guess the next position
		i := last.Framesize() - 4
		if l < (i + 4) {
			// HACK: the guess position isn't available yet - wait for the next read
			// to prevent a false positive in the slow path
			return -1, last
		}

		rv := mp3header{buf[i], buf[i+1], buf[i+2], buf[i+3]}
		if rv.SyncWord() == 0xffe && last.B == rv.B && last.SampleRate() == rv.SampleRate() {
			// We're in luck!
			return i, rv
		}
	}

	// Short path isn't available - scan the full buffer
	for i, c := range buf[:l-4] {
		if c != 0xff {
			continue
		}

		rv := mp3header{buf[i], buf[i+1], buf[i+2], buf[i+3]}
		if rv.SyncWord() != 0xffe || rv.Bitrate() == 0 || rv.SampleRate() == 0 {
			continue
		}

		if last.A != 0xff {
			// The previous header was empty - this one's good
			return i, rv
		} else {
			// Check if key fields match the previous header
			if last.B == rv.B && last.SampleRate() == rv.SampleRate() {
				return i, rv
			}
		}
	}

	return -1, last
}
