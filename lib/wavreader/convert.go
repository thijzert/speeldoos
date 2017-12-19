package wavreader

import (
	"fmt"
	"io"
)

func Convert(r *Reader, channels, rate, bits int) (*Reader, error) {
	// Fast path: don't convert anything if not absolutely necessary
	if r.Channels == channels && r.SampleRate == rate && r.BitsPerSample == bits {
		//return r, nil
	}

	if channels < 1 {
		return nil, fmt.Errorf("need at least 1 output channel")
	}

	rv, wri := Pipe()
	rv.Channels = channels
	rv.SampleRate = rate
	rv.BitsPerSample = bits

	go doConversion(wri, r, channels, rate, bits)

	return rv, nil
}

func doConversion(wri *io.PipeWriter, r *Reader, channels, rate, bits int) {
	// Samples at a time. We're reading the input stream in chunks of, say, 10ms.
	saatIn, saatOut := (r.SampleRate+99)/100, 1+(rate+99)/100

	// Bytes per sample
	Bin, Bout := (r.BitsPerSample+7)/8, (bits+7)/8

	// Create buffers
	bufIn := make([]byte, r.Channels*saatIn*Bin)
	bufChan := make([][]byte, channels)
	bufRate := make([]*rateConverter, channels)
	bufBits := make([][]byte, channels)
	bufOut := make([]byte, r.Channels*saatOut*Bout)

	for i, _ := range bufChan {
		bufChan[i] = make([]byte, saatIn*Bin)

		if r.SampleRate == rate {
			// Optimization: since we're not converting, just reuse the last output buffer
			bufRate[i] = &rateConverter{Output: bufChan[i]}
		} else {
			bufRate[i] = &rateConverter{Output: make([]byte, saatOut*Bin)}
		}

		if r.BitsPerSample == bits {
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
			nRate, err = rc.convert(bufChan[i][:nMono], r, Bin, rate)
			if err != nil {
				wri.CloseWithError(err)
				return
			}
		}

		for i, ch := range bufRate {
			nBits, err = convertBits(bufBits[i], ch.Output[:nRate], r, Bin, Bout, bits)
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
		if errRead != nil {
			wri.CloseWithError(errRead)
			return
		}
	}
}

func monoChannels(out [][]byte, in []byte, r *Reader, Bin int) (int, error) {
	if r.Channels == 1 {
		// Copy mono input data to all output channels
		for _, b := range out {
			copy(b, in)
		}
		return len(in), nil
	} else if r.Channels == len(out) {
		// Spice interleaved sample data into per-channel buffers
		j := 0
		for j*r.Channels < len(in) {
			for i, ch := range out {
				ioff := j*r.Channels + i*Bin
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
	Output  []byte
	skipped int
}

func (rc *rateConverter) convert(in []byte, r *Reader, Bin, rate int) (int, error) {
	if r.SampleRate == rate {
		// We've caught this case by reusing buffers
		return len(in), nil
	}

	if r.SampleRate > rate && (r.SampleRate%rate) == 0 {
		// Fast path: the source sample rate is a multiple of the target rate

		// Output a sample every c samples
		c := r.SampleRate / rate

		i, n := 0, 0

		for i < len(in) {
			if rc.skipped == 0 {
				copy(rc.Output[n:], in[i:i+Bin])
				n += Bin
			}

			rc.skipped = (rc.skipped + 1) % c
			i += Bin
		}

		return n, nil
	} else if r.SampleRate < rate && (rate%r.SampleRate) == 0 {
		// Another fast path: approximate the upscaled version with squarewaves

		// Repeat each sample c times
		c := rate / r.SampleRate

		i := 0
		for i < len(in) {
			for j := 0; j < c; j++ {
				copy(rc.Output[i*c+j*Bin:], in[i:i+Bin])
			}
			i += Bin
		}

		return len(in) * c, nil
	}

	return 0, fmt.Errorf("sample rate conversion is not implemented")
}

func convertBits(out []byte, in []byte, r *Reader, Bin, Bout, bits int) (int, error) {
	if r.BitsPerSample == bits {
		// We've caught this case by reusing buffers
		return len(in), nil
	}

	// FIXME: handle bit lengths that aren't multiples of 8 bits. (Do those exist?)

	if r.BitsPerSample > bits {
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
