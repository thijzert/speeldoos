package speeldoos

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/zipmap"
)

type Library struct {
	LibraryDir string
	WAVConf    wavreader.Config
	Carriers   []ParsedCarrier
	zip        *zipmap.ZipMap
}

type ParsedCarrier struct {
	Filename string
	Carrier  *Carrier
	Error    error
}

func NewLibrary(dir string) *Library {
	rv := &Library{
		LibraryDir: dir,
		zip:        zipmap.New(),
	}
	return rv
}

func (l *Library) Refresh() error {
	rv := []ParsedCarrier{}

	d, err := os.Open(l.LibraryDir)
	if err != nil {
		return err
	}

	files, err := d.Readdir(0)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fn := f.Name()
		if len(fn) < 5 || fn[len(fn)-4:] != ".xml" {
			continue
		}

		pc := ParsedCarrier{Filename: path.Join(l.LibraryDir, fn)}
		pc.Carrier, pc.Error = ImportCarrier(pc.Filename)

		rv = append(rv, pc)
	}

	l.Carriers = rv
	return nil
}

func (l *Library) AllCarriers() []ParsedCarrier {
	rv := make([]ParsedCarrier, 0, len(l.Carriers))
	for _, pc := range l.Carriers {
		if pc.Error == nil {
			rv = append(rv, pc)
		}
	}
	return rv
}

func (l *Library) GetWAV(pf Performance) (*wavreader.Reader, error) {
	ch, rate, bits, bps := 0, 0, 0, 0
	fixedSize := 0
	for i, f := range pf.SourceFiles {
		fl, er := l.zip.Get(path.Join(l.LibraryDir, f.Filename))
		if er != nil {
			return nil, er
		}
		defer fl.Close()

		ww, er := l.WAVConf.FromFLAC(fl)
		if er != nil {
			return nil, er
		}
		defer ww.Close()
		ww.Init()

		if i == 0 {
			ch, rate, bits = ww.Channels, ww.SampleRate, ww.BitsPerSample
			bps = ch * ((bits + 7) / 8)
		} else if ch != ww.Channels || rate != ww.SampleRate || bits != ww.BitsPerSample {
			return nil, fmt.Errorf("audio format mismatch: part %d has %d channels, %d bits at %dHz; previously it was %d channels, %d bits at %dHz", i+1, ww.Channels, ww.BitsPerSample, ww.SampleRate, ch, bits, rate)
		}

		if ww.Size == 0 || (ww.Size%bps) != 0 {
			return nil, fmt.Errorf("wav length (%d) is not a multiple of bytes per sample (%d)", ww.Size, bps)
		}
		fixedSize += ww.Size
	}

	rv, wri := wavreader.Pipe()
	rv.Channels, rv.SampleRate, rv.BitsPerSample = ch, rate, bits
	rv.Size = fixedSize

	go func() {
		for _, f := range pf.SourceFiles {
			fl, er := l.zip.Get(path.Join(l.LibraryDir, f.Filename))
			if er != nil {
				wri.CloseWithError(er)
			}
			defer fl.Close()

			ww, er := l.WAVConf.FromFLAC(fl)
			if er != nil {
				wri.CloseWithError(er)
			}
			ww.Init()
			defer ww.Close()

			_, er = io.Copy(wri, ww)
			if er != nil {
				wri.CloseWithError(er)
			}
		}

		wri.Close()
	}()
	return rv, nil
}
