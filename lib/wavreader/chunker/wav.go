package chunker

import (
	"io"
	"log"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
)

type WAVChunkConfig struct {
	StreamFormat wavreader.StreamFormat
	ReadAhead    time.Duration
	ReadBehind   time.Duration
}

func (c WAVChunkConfig) New() (Chunker, error) {
	// Use sensible defaults if the RAH/RBH aren't set
	if c.ReadAhead < 1*time.Millisecond {
		c.ReadAhead = 30 * time.Second
	}
	if c.ReadBehind < 1*time.Millisecond {
		c.ReadBehind = 15 * time.Second
	}

	nChunks := 50

	// How long should each chunk be?
	chunkDuration := 1 * time.Second
	samplesPerChunk := c.StreamFormat.Rate
	if c.StreamFormat.Rate%100 == 0 {
		// The sample rate is divisible by 100 - use 10ms chunks
		chunkDuration = 10 * time.Millisecond
		samplesPerChunk = c.StreamFormat.Rate / 100

		nChunks = int((c.ReadAhead+c.ReadBehind)/(10*time.Millisecond)) + 2
	}

	bytesPerSample := c.StreamFormat.BytesPerSample()

	log.Printf("WAV chunker reading %s ahead; with %d chunks of length %d bytes each", c.ReadAhead, nChunks, samplesPerChunk*bytesPerSample)

	rv := &wavChunker{
		streamFormat: c.StreamFormat,
		embargo:      time.Now(),
		partialChunk: make([]byte, 0, samplesPerChunk*bytesPerSample),
		chcont: &chunkContainer{
			chunks: make([]chunk, nChunks),
			start:  0,
			end:    0,
		},
		chunkDuration: chunkDuration,
		readAhead:     c.ReadAhead,
	}

	return rv, nil
}

type wavChunker struct {
	streamFormat  wavreader.StreamFormat
	embargo       time.Time
	partialChunk  []byte
	chcont        *chunkContainer
	chunkDuration time.Duration
	readAhead     time.Duration
}

func (m *wavChunker) Init(fixedSize int) error {
	if m.chcont.errorState != nil {
		return m.chcont.errorState
	}
	return nil
}

func (m *wavChunker) Format() wavreader.StreamFormat {
	return m.streamFormat
}

func (m *wavChunker) Write(buf []byte) (int, error) {
	if m.chcont.errorState != nil {
		return 0, m.chcont.errorState
	}

	var err error

	i := len(m.partialChunk)
	n := len(buf)

	if i+n > cap(m.partialChunk) {
		n = cap(m.partialChunk) - i
	}

	m.partialChunk = m.partialChunk[:i+n]
	copy(m.partialChunk[i:], buf[:n])

	// If the chunk's full, add it to the pile
	if i+n == cap(m.partialChunk) {
		err = m.admitFullChunk()
		if err != nil {
			return 0, err
		}
	}

	i = 0
	if n < len(buf) {
		i, err = m.Write(buf[n:])
	}
	return n + i, err
}

func (m *wavChunker) admitFullChunk() error {
	for time.Now().Add(m.readAhead).Before(m.embargo) {
		time.Sleep(1 * time.Millisecond)
	}

	m.chcont.AddChunk(m.partialChunk, m.embargo)
	m.embargo = m.embargo.Add(m.chunkDuration)
	m.partialChunk = m.partialChunk[:0]
	return nil
}

func (m *wavChunker) Close() error {
	if m.chcont.errorState != nil {
		return m.chcont.errorState
	}
	m.chcont.errorState = io.EOF
	return nil
}

func (m *wavChunker) CloseWithError(er error) error {
	if m.chcont.errorState != nil {
		return m.chcont.errorState
	}
	m.chcont.errorState = er
	return er
}

func (m *wavChunker) NewStream() (ChunkStream, error) {
	return m.chcont.NewStream()
}

func (m *wavChunker) NewStreamWithOffset(offset time.Duration) (ChunkStream, error) {
	return m.chcont.NewStreamWithOffset(offset)
}

func (m *wavChunker) BufferStatus() BufferStatus {
	return m.chcont.BufferStatus()
}

func (m *wavChunker) SetAssociatedData(data interface{}) {
	m.chcont.SetAssociatedData(data)
}

func (m *wavChunker) GetAssociatedData() (interface{}, error) {
	return m.chcont.GetAssociatedData()
}
