package chunker

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
)

// A Chunker is a buffered writer that breaks up audio into chunks made available for reading later.
type Chunker interface {
	wavreader.Writer
	NewStream() (ChunkStream, error)
}

// A ChunkStream wraps a single read session initiated from a Chunker
type ChunkStream interface {
	io.Reader
}

type chunk struct {
	contents []byte
	embargo  time.Time
}

type mp3Chunker struct {
	audioIn wavreader.Writer
	mp3out  *io.PipeReader

	mu         sync.RWMutex
	errorState error
	chunks     []chunk
	start, end int
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
		chunks:  make([]chunk, 30),
		start:   0,
		end:     0,
	}

	go rv.splitChunks()

	return rv, nil
}

func (m *mp3Chunker) Init(fixedSize int) error {
	if m.errorState != nil {
		return m.errorState
	}
	return m.audioIn.Init(fixedSize)
}
func (m *mp3Chunker) Format() wavreader.StreamFormat {
	return m.audioIn.Format()
}
func (m *mp3Chunker) Write(buf []byte) (int, error) {
	if m.errorState != nil {
		return 0, m.errorState
	}
	return m.audioIn.Write(buf)
}
func (m *mp3Chunker) Close() error {
	if m.errorState != nil {
		return m.errorState
	}
	m.errorState = io.EOF
	return m.audioIn.Close()
}
func (m *mp3Chunker) CloseWithError(er error) error {
	if m.errorState != nil {
		return m.errorState
	}
	m.errorState = er
	return m.audioIn.CloseWithError(er)
}

func (m *mp3Chunker) splitChunks() {
	buf := make([]byte, 1600)
	embargo := time.Now()
	var err error
	var n int
	for {
		n, err = m.mp3out.Read(buf)
		if err != nil {
			log.Printf("splitting hairs got me this error: %s", err)
			break
		}

		if n == 0 {
			continue
		}

		chbuf := make([]byte, n)
		copy(chbuf, buf[:n])

		m.mu.Lock()
		l := len(m.chunks)
		next := m.end
		m.end = (m.end + 1 + 2*l) % l
		m.start = (m.end + 5 + 2*l) % l
		m.chunks[next].embargo = embargo
		m.chunks[next].contents = chbuf
		m.mu.Unlock()

		embargo = embargo.Add(20 * time.Millisecond)
		for time.Now().Add(100 * time.Millisecond).Before(embargo) {
			time.Sleep(1 * time.Millisecond)
		}
	}
	m.CloseWithError(err)
}

func (m *mp3Chunker) NewStream() (ChunkStream, error) {
	if m.errorState != nil {
		return nil, m.errorState
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	start := m.start
	for start != m.end {
		if m.chunks[start].embargo.Before(now) {
			break
		}
		start = (start + 1) % len(m.chunks)
	}

	return &chunkReader{
		parent:  m,
		current: start,
	}, nil
}

type chunkReader struct {
	parent   *mp3Chunker
	current  int
	embargo  time.Time
	buf      []byte
	bufindex int
}

func (ch *chunkReader) readBuffer(b []byte) (n int) {
	for time.Now().Before(ch.embargo) {
		time.Sleep(1 * time.Millisecond)
	}

	if ch.buf == nil || len(ch.buf)-ch.bufindex <= 0 {
		return 0
	}

	l := ch.bufindex + len(b)
	if l > len(ch.buf) {
		l = len(ch.buf)
	}

	copy(b, ch.buf[ch.bufindex:l])
	n += l - ch.bufindex
	if l < len(ch.buf) {
		ch.bufindex = l
	} else {
		ch.buf = nil
		ch.bufindex = 0
	}
	return n
}

func (ch *chunkReader) getNext() bool {
	ch.parent.mu.RLock()
	defer ch.parent.mu.RUnlock()

	next := ch.current + 1
	if next == len(ch.parent.chunks) {
		next = 0
	}
	if ch.parent.start <= ch.parent.end {
		if next < ch.parent.start || next >= ch.parent.end {
			return false
		}
	} else {
		if next < ch.parent.start && next >= ch.parent.end {
			return false
		}
	}
	ch.current = next

	cch := ch.parent.chunks[ch.current]
	ch.embargo = cch.embargo
	ch.buf = make([]byte, len(cch.contents))
	copy(ch.buf, cch.contents)
	ch.bufindex = 0
	return true
}

func (ch *chunkReader) Read(b []byte) (n int, err error) {
	n += ch.readBuffer(b)
	if n > 0 {
		return
	}

	if !ch.getNext() {
		if ch.parent.errorState != nil {
			return 0, ch.parent.errorState
		}

		// HACK: the writer isn't ready yet, but we want to avoid effectively spinlooping
		time.Sleep(1 * time.Millisecond)
		return 0, nil
	}

	n += ch.readBuffer(b)
	return
}
