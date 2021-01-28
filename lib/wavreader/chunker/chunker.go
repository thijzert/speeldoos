package chunker

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
)

// A Chunker is a buffered writer that breaks up audio into chunks made available for reading later.
type Chunker interface {
	wavreader.Writer
	NewStream() (ChunkStream, error)
	NewStreamWithOffset(time.Duration) (ChunkStream, error)
}

// A ChunkStream wraps a single read session initiated from a Chunker
type ChunkStream interface {
	io.Reader
}

type BufferStatus struct {
	// Tmin and Tmax denote the lowest and highest wall clock time in the buffer
	Tmin, Tmax time.Time

	// Tahead is the length of the buffer (in milliseconds) added but not yet available for reading
	Tahead float32

	// Tbehind is the length of the buffer (in milliseconds) available for reading as of the current time
	Tbehind float32

	// BufferSize is the size (in bytes) added but not yet available for reading
	BufferSize int

	// TotalSize is the total size (in bytes) of the buffer
	TotalSize int
}

type Statuser interface {
	BufferStatus() BufferStatus
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

func (chcont *chunkContainer) NewStream() (ChunkStream, error) {
	return chcont.NewStreamWithOffset(0)
}

func (chcont *chunkContainer) NewStreamWithOffset(offset time.Duration) (ChunkStream, error) {
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
		offset:  offset,
	}, nil
}

func (chcont *chunkContainer) AddChunk(buf []byte, embargo time.Time) {
	chbuf := make([]byte, len(buf))
	copy(chbuf, buf)

	chcont.mu.Lock()
	l := len(chcont.chunks)
	next := chcont.end
	chcont.end = (next + 1 + 2*l) % l
	if next < chcont.start || next+5 > l {
		chcont.start = (next + 5 + 2*l) % l
	}
	chcont.chunks[next].embargo = embargo
	chcont.chunks[next].contents = chbuf
	chcont.chunks[next].seqno = chcont.seqno
	chcont.seqno++
	chcont.mu.Unlock()
}

func (chcont *chunkContainer) BufferStatus() BufferStatus {
	var rv BufferStatus

	chcont.mu.RLock()
	defer chcont.mu.RUnlock()

	nowish := time.Now()

	appendChunkStatus := func(i int) {
		chunk := chcont.chunks[i]

		if rv.Tmin.IsZero() {
			rv.Tmin = chunk.embargo
		}

		if nowish.After(chunk.embargo) {
			rv.BufferSize += len(chunk.contents)
		}
		rv.TotalSize += len(chunk.contents)

		rv.Tmax = chunk.embargo
	}

	if chcont.start < chcont.end {
		for i := chcont.start; i < chcont.end; i++ {
			appendChunkStatus(i)
		}
	} else {
		for i := chcont.start; i < len(chcont.chunks); i++ {
			appendChunkStatus(i)
		}
		for i := 0; i < chcont.end; i++ {
			appendChunkStatus(i)
		}
	}

	rv.Tbehind = float32(1000.0 * nowish.Sub(rv.Tmin).Seconds())
	rv.Tahead = float32(1000.0 * rv.Tmax.Sub(nowish).Seconds())

	return rv
}

type chunkReader struct {
	parent   *chunkContainer
	current  int
	embargo  time.Time
	buf      []byte
	bufindex int
	seqno    uint32
	offset   time.Duration
}

func (ch *chunkReader) readBuffer(b []byte) (n int) {
	for time.Now().Add(ch.offset).Before(ch.embargo) {
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
