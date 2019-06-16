package chunker

import (
	"io"
	"log"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
)

type mp3Chunker struct {
	audioIn wavreader.Writer
	mp3out  *io.PipeReader
	embargo time.Time
	chcont  *chunkContainer
}

func NewMP3() (Chunker, error) {
	r, w := io.Pipe()

	wavin, err := wavreader.ToMP3(w, wavreader.DAT)
	if err != nil {
		return nil, err
	}

	rv := &mp3Chunker{
		audioIn: wavin,
		mp3out:  r,
		embargo: time.Now(),
		chcont: &chunkContainer{
			chunks: make([]chunk, 30),
			start:  0,
			end:    0,
		},
	}

	go rv.splitChunks()

	return rv, nil
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

func (m *mp3Chunker) splitChunks() {
	buf := make([]byte, 1600)
	var err error
	var n int
	for {
		n, err = m.mp3out.Read(buf)
		if err != nil {
			if err != io.EOF {
				// TODO: remove
				log.Printf("splitting hairs got me this error: %s", err)
			}
			break
		}

		if n == 0 {
			continue
		}

		m.chcont.AddChunk(buf[:n], m.embargo)

		m.embargo = m.embargo.Add(20 * time.Millisecond)
		for time.Now().Add(100 * time.Millisecond).Before(m.embargo) {
			time.Sleep(1 * time.Millisecond)
		}
	}
	m.CloseWithError(err)
}

func (m *mp3Chunker) NewStream() (ChunkStream, error) {
	return m.chcont.NewStream()
}
