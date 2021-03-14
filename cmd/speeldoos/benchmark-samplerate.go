package main

import (
	"io"
	"log"
	"math"
	"os"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
)

func benchmark_samplerate(args []string) {
	fIn := wavreader.StreamFormat{Format: 1, Channels: 1, Rate: 44100, Bits: 24}
	fOut := wavreader.StreamFormat{Format: 1, Channels: 1, Rate: 48000, Bits: 24}

	rd := &repeatReader{
		StreamFormat: fIn,
	}

	if len(rd.Samples) < 2 {
		// Create a 441Hz sine wave (i.e. a sine wave exactly 100 samples long)
		rd.Samples = make([]uint32, 100)
		invtmax := (2.0 * math.Pi) / float64(len(rd.Samples))

		for i := range rd.Samples {
			t := float64(i) * invtmax
			v := int32(0.5 + float64(0x3fffff)*math.Sin(t))
			rd.Samples[i] = uint32(v)
		}
	}

	// The input sample is 100 seconds long
	const nSeconds = 120
	rd.SetSize(fIn.BytesPerSample() * fIn.Rate * nSeconds)

	var err error
	var wOut io.Writer = io.Discard

	defer func() {
		if wc, ok := wOut.(io.WriteCloser); ok {
			wc.Close()
		}
	}()

	for _, fileName := range args {
		if len(fileName) > 5 && fileName[len(fileName)-4:] == ".wav" {
			wOut, err = os.Create(fileName)
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}

	wavOut := wavreader.NewWriter(wOut, fOut)

	log.Printf("Starting conversion")

	wavIn, err := wavreader.Convert(rd, fOut)
	if err != nil || wavIn == nil {
		log.Fatal(err)
	}

	t0 := time.Now()
	_, err = io.Copy(wavOut, wavIn)
	t1 := time.Now()

	if err != nil {
		log.Fatal(err)
	}

	d := t1.Sub(t0)
	log.Printf("Conversion done. Time taken: %s", d)

	d1 := nSeconds * time.Second
	speed := float64(d1.Microseconds()) / float64(d.Microseconds())
	log.Printf("This device can convert from %gkHz to %gkHz at %.1f× speed", 0.001*float64(fIn.Rate), 0.001*float64(fOut.Rate), speed)
}

type repeatReader struct {
	Samples      []uint32
	StreamFormat wavreader.StreamFormat
	SamplesRead  int
	TotalSize    int
}

func (f *repeatReader) Close() error {
	return nil
}

func (f *repeatReader) Read(buf []byte) (int, error) {
	if f.SamplesRead >= f.TotalSize {
		return 0, io.EOF
	}

	bps := f.StreamFormat.BytesPerSample()
	samples := len(buf) / bps
	if (f.SamplesRead + samples) > f.TotalSize {
		samples = f.TotalSize - f.SamplesRead
	}

	modn := len(f.Samples)
	sidx := f.SamplesRead % len(f.Samples)

	for i := 0; i < samples; i++ {
		n := f.Samples[sidx]
		sidx++
		if sidx == modn {
			sidx = 0
		}

		itoa(buf[bps*i:bps*i+bps], n)
	}

	f.SamplesRead += samples
	return samples * bps, nil
}

func (f *repeatReader) Format() wavreader.StreamFormat {
	return f.StreamFormat
}

func (f *repeatReader) Init() {
	f.SamplesRead = 0
}

func (f *repeatReader) Size() int {
	return f.TotalSize * f.StreamFormat.BytesPerSample()
}

func (f *repeatReader) SetSize(s int) {
	f.TotalSize = s / f.StreamFormat.BytesPerSample()
}

func itoa(buf []byte, v uint32) {
	for i := range buf {
		buf[i] = byte(v & 0xff)
		v >>= 8
	}
}
