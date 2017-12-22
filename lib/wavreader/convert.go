package wavreader

import (
	"fmt"
	"io"
)

const (
	// The length, in milliseconds, of conversion chunks
	msCHUNK = 10
)

func Convert(r *Reader, format StreamFormat) (*Reader, error) {
	// Fast path: don't convert anything if not absolutely necessary
	if r.Format == format {
		return r, nil
	}

	if format.Channels < 1 {
		return nil, fmt.Errorf("need at least 1 output channel")
	}

	rv, wri := Pipe()
	rv.Format = format

	go doConversion(wri, r, format)

	return rv, nil
}

func doConversion(wri *io.PipeWriter, r *Reader, format StreamFormat) {
	// Samples at a time. We're reading the input stream in chunks of, say, 10ms.
	saatIn, saatOut := (msCHUNK*r.Format.Rate+999)/1000, 1+(msCHUNK*format.Rate+999)/1000

	// Bytes per sample
	Bin, Bout := (r.Format.Bits+7)/8, (format.Bits+7)/8

	// Create buffers
	bufIn := make([]byte, r.Format.Channels*saatIn*Bin)
	bufChan := make([][]byte, format.Channels)
	bufRate := make([]*rateConverter, format.Channels)
	bufBits := make([][]byte, format.Channels)
	bufOut := make([]byte, format.Channels*saatOut*Bout)

	for i, _ := range bufChan {
		bufChan[i] = make([]byte, saatIn*Bin)

		bufRate[i] = newRateConverter(r, format.Rate, bufChan[i])

		if r.Format.Bits == format.Bits {
			// Optimization: since we're not converting, just reuse the last output buffer
			bufBits[i] = bufRate[i].Output
		} else {
			bufBits[i] = make([]byte, saatOut*Bout)
		}
	}

	var err error
	var nMono, nRate, nBits int

	for {
		n, errRead := io.ReadFull(r, bufIn)
		if n == 0 && errRead != nil {
			wri.CloseWithError(errRead)
			return
		}

		nMono, err = monoChannels(bufChan, bufIn[:n], r, Bin)
		if err != nil {
			wri.CloseWithError(err)
			return
		}

		for i, rc := range bufRate {
			nRate, err = rc.convert(bufChan[i][:nMono])
			if err != nil {
				wri.CloseWithError(err)
				return
			}
		}

		for i, ch := range bufRate {
			nBits, err = convertBits(bufBits[i], ch.Output[:nRate], r, Bin, Bout, format.Bits)
			if err != nil {
				wri.CloseWithError(err)
				return
			}
		}

		n, err = interleave(bufOut, bufBits, nBits, Bout)
		if err != nil {
			wri.CloseWithError(err)
			return
		}

		_, errWrite := wri.Write(bufOut[:n])
		if errWrite != nil {
			wri.CloseWithError(errWrite)
			return
		}
	}
}

func monoChannels(out [][]byte, in []byte, r *Reader, Bin int) (int, error) {
	if r.Format.Channels == 1 {
		// Copy mono input data to all output channels
		for _, b := range out {
			copy(b, in)
		}
		return len(in), nil
	} else if r.Format.Channels == len(out) {
		// Spice interleaved sample data into per-channel buffers
		j := 0
		for j*r.Format.Channels < len(in) {
			for i, ch := range out {
				ioff := j*r.Format.Channels + i*Bin
				copy(ch[j:], in[ioff:ioff+Bin])
			}
			j += Bin
		}
		return j, nil
	} else {
		return 0, fmt.Errorf("n:m channel mapping is not implemented")
	}
}

type rateConverter struct {
	Output          []byte
	skipped         int
	rateIn, rateOut int
	bin             int
	saatIn, saatOut int
}

func newRateConverter(r *Reader, rate int, input []byte) *rateConverter {
	rc := &rateConverter{
		bin:     (r.Format.Bits + 7) / 8,
		rateIn:  r.Format.Rate,
		rateOut: rate,
		saatIn:  (msCHUNK*r.Format.Rate + 999) / 1000,
		saatOut: 1 + (msCHUNK*rate+999)/1000,
	}

	if rc.rateIn == rc.rateOut {
		// Optimization: since we're not converting, just reuse the input buffer as output
		rc.Output = input
	} else {
		rc.Output = make([]byte, rc.saatOut*rc.bin)
	}

	return rc
}

func (rc *rateConverter) convert(in []byte) (int, error) {
	if rc.rateIn == rc.rateOut {
		// We've caught this case by reusing buffers
		return len(in), nil
	}

	if rc.rateIn > rc.rateOut && (rc.rateIn%rc.rateOut) == 0 {
		// Fast path: the source sample rate is a multiple of the target rate

		// Output a sample every c samples
		c := rc.rateIn / rc.rateOut

		i, n := 0, 0

		for i < len(in) {
			if rc.skipped == 0 {
				copy(rc.Output[n:], in[i:i+rc.bin])
				n += rc.bin
			}

			rc.skipped = (rc.skipped + 1) % c
			i += rc.bin
		}

		return n, nil
	} else if rc.rateIn < rc.rateOut && (rc.rateOut%rc.rateIn) == 0 {
		// Another fast path: approximate the upscaled version with squarewaves

		// Repeat each sample c times
		c := rc.rateOut / rc.rateIn

		i := 0
		for i < len(in) {
			for j := 0; j < c; j++ {
				copy(rc.Output[i*c+j*rc.bin:], in[i:i+rc.bin])
			}
			i += rc.bin
		}

		return len(in) * c, nil
	}

	return 0, fmt.Errorf("sample rate conversion is not implemented")
}

func convertBits(out []byte, in []byte, r *Reader, Bin, Bout, bits int) (int, error) {
	if r.Format.Bits == bits {
		// We've caught this case by reusing buffers
		return len(in), nil
	}

	// FIXME: handle bit lengths that aren't multiples of 8 bits. (Do those exist?)

	if r.Format.Bits > bits {
		// Discard input bits in the output stream
		i, n := 0, 0

		for i < len(in) {
			// HACK: Handle conversion from signed 16-bit to unsigned 8-bit
			if bits == 8 {
				out[n] = 128 + in[i+Bin-1]
			} else {
				copy(out[n:], in[i+Bin-Bout:i+Bin])
			}

			i += Bin
			n += Bout
		}

		return n, nil
	} else {
		// Pad the input stream to fill the output stream
		i, n := 0, 0

		// HACK: Handle conversion from unsigned 8-bit to signed 16-bit
		var offset uint8 = 0
		if Bin == 1 {
			offset = 128
		}

		for i < len(in) {
			for j := 0; j < Bout; j++ {
				out[n+Bout-j-1] = in[i+(Bin-(j%Bin)-1)] + offset
				// TODO: Add dithering, or some other method of hiding rounding errors
			}

			i += Bin
			n += Bout
		}

		return n, nil
	}
}

func interleave(out []byte, in [][]byte, length int, Bout int) (int, error) {
	c := len(in)
	j := 0
	for j < length {
		for i, ch := range in {
			ooff := j*c + i*Bout
			copy(out[ooff:], ch[j:j+Bout])
		}
		j += Bout
	}
	return j * c, nil
}
