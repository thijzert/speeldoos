package chunker

import (
	"fmt"
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
	seqno    uint32
}

type chunkContainer struct {
	mu         sync.RWMutex
	errorState error
	chunks     []chunk
	start, end int
	seqno      uint32
}

type mp3Chunker struct {
	audioIn wavreader.Writer
	mp3out  *io.PipeReader
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

		m.chcont.AddChunk(buf[:n], embargo)

		embargo = embargo.Add(20 * time.Millisecond)
		for time.Now().Add(100 * time.Millisecond).Before(embargo) {
			time.Sleep(1 * time.Millisecond)
		}
	}
	m.CloseWithError(err)
}

func (m *mp3Chunker) NewStream() (ChunkStream, error) {
	return m.chcont.NewStream()
}

func (chcont *chunkContainer) NewStream() (ChunkStream, error) {
	if chcont.errorState != nil {
		return nil, chcont.errorState
	}

	chcont.mu.RLock()
	defer chcont.mu.RUnlock()

	now := time.Now()
	start := chcont.start
	for start != chcont.end {
		if chcont.chunks[start].embargo.Before(now) {
			break
		}
		start = (start + 1) % len(chcont.chunks)
	}

	return &chunkReader{
		parent:  chcont,
		current: start,
		seqno:   chcont.chunks[start].seqno,
	}, nil
}

func (chcont *chunkContainer) AddChunk(buf []byte, embargo time.Time) {
	chbuf := make([]byte, len(buf))
	copy(chbuf, buf)

	chcont.mu.Lock()
	l := len(chcont.chunks)
	next := chcont.end
	chcont.end = (chcont.end + 1 + 2*l) % l
	chcont.start = (chcont.end + 5 + 2*l) % l
	chcont.chunks[next].embargo = embargo
	chcont.chunks[next].contents = chbuf
	chcont.chunks[next].seqno = chcont.seqno
	chcont.seqno++
	chcont.mu.Unlock()
}

type chunkReader struct {
	parent   *chunkContainer
	current  int
	embargo  time.Time
	buf      []byte
	bufindex int
	seqno    uint32
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

func (ch *chunkReader) getNext() (bool, error) {
	ch.parent.mu.RLock()
	defer ch.parent.mu.RUnlock()

	next := ch.current + 1
	if next == len(ch.parent.chunks) {
		next = 0
	}
	if ch.parent.start <= ch.parent.end {
		if next < ch.parent.start || next >= ch.parent.end {
			return false, nil
		}
	} else {
		if next < ch.parent.start && next >= ch.parent.end {
			return false, nil
		}
	}
	ch.current = next

	cch := ch.parent.chunks[ch.current]
	ch.seqno++
	if ch.seqno != cch.seqno {
		return false, fmt.Errorf("sequence number mismatch")
	}
	ch.embargo = cch.embargo
	ch.buf = make([]byte, len(cch.contents))
	copy(ch.buf, cch.contents)
	ch.bufindex = 0
	return true, nil
}

func (ch *chunkReader) Read(b []byte) (n int, err error) {
	n += ch.readBuffer(b)
	if n > 0 {
		return
	}

	var changed bool
	changed, err = ch.getNext()
	if err != nil {
		return
	}

	if !changed {
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
