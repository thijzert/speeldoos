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
	bufRate := make([][]byte, channels)
	bufBits := make([][]byte, channels)
	bufOut := make([]byte, r.Channels*saatOut*Bout)

	for i, _ := range bufChan {
		bufChan[i] = make([]byte, saatIn*Bin)

		if r.SampleRate == rate {
			// Optimization: since we're not converting, just reuse the last output buffer
			bufRate[i] = bufChan[i]
		} else {
			bufRate[i] = make([]byte, saatOut*Bin)
		}

		if r.BitsPerSample == bits {
			// Optimization: since we're not converting, just reuse the last output buffer
			bufBits[i] = bufRate[i]
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

		for i, ch := range bufChan {
			nRate, err = convertRate(bufRate[i], ch[:nMono], r, Bin, rate)
			if err != nil {
				wri.CloseWithError(err)
				return
			}
		}

		for i, ch := range bufRate {
			nBits, err = convertBits(bufBits[i], ch[:nRate], r, Bin, bits)
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

func convertRate(out []byte, in []byte, r *Reader, Bin, rate int) (int, error) {
	if r.SampleRate == rate {
		// We've caught this case by reusing buffers
		return len(in), nil
	}

	return 0, fmt.Errorf("sample rate conversion is not implemented")
}

func convertBits(out []byte, in []byte, r *Reader, Bin, bits int) (int, error) {
	if r.BitsPerSample == bits {
		// We've caught this case by reusing buffers
		return len(in), nil
	}

	return 0, fmt.Errorf("bit resolution conversion is not implemented")
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
