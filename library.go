package speeldoos

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/zipmap"
)

// A collection of Carriers
type Library struct {
	LibraryDir string
	WAVConf    wavreader.Config
	Carriers   []ParsedCarrier
	zip        *zipmap.ZipMap
}

type ParsedCarrier struct {
	// The full path to the xml file
	Filename string

	// The parsed Carrier, if successful
	Carrier *Carrier

	// The parse error, if applicable
	Error error
}

func NewLibrary(dir string) *Library {
	rv := &Library{
		LibraryDir: dir,
		zip:        zipmap.New(),
	}
	return rv
}

// (Re-)read all XML files from disk, parsing any speeldoos files
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

// Filter all Carriers in the library that are error-free
func (l *Library) AllCarriers() []ParsedCarrier {
	rv := make([]ParsedCarrier, 0, len(l.Carriers))
	for _, pc := range l.Carriers {
		if pc.Error == nil {
			rv = append(rv, pc)
		}
	}
	return rv
}

// Open one performance in the library and return its raw audio data
func (l *Library) GetWAV(pf Performance) (*wavreader.Reader, error) {
	var format wavreader.StreamFormat
	bps := 0
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
		ww.Init()
		ww.Close()

		if i == 0 {
			format = ww.Format
			bps = format.Channels * ((format.Bits + 7) / 8)
		} else if ww.Format != format {
			return nil, fmt.Errorf("audio format mismatch: part %d has %d channels, %dHz, %d bits; previously it was %d channels, %dHz, %d bits", i+1,
				ww.Format.Channels, ww.Format.Rate, ww.Format.Bits,
				format.Channels, format.Rate, format.Bits)
		}

		if ww.Size == 0 || (ww.Size%bps) != 0 {
			return nil, fmt.Errorf("wav length (%d) is not a multiple of bytes per sample (%d)", ww.Size, bps)
		}
		fixedSize += ww.Size
	}

	rv, wri := wavreader.Pipe()
	rv.Format = format
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

			_, er = io.Copy(wri, ww)
			if er != nil {
				wri.CloseWithError(er)
			}

			ww.Close()
		}

		wri.Close()
	}()
	return rv, nil
}
