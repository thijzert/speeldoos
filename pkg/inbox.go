package pkg

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

type detectedFile struct {
	Path      string
	Parts     []string
	Extension string
	Disc      int
}

func detectFile(filename string) detectedFile {
	rv := detectedFile{
		Path:  filename,
		Parts: strings.Split(filename, "/"),
	}

	exts := strings.Split(rv.Parts[len(rv.Parts)-1], ".")
	if len(exts) > 1 {
		rv.Extension = exts[len(exts)-1]
	}

	return rv
}

type preliminaryCarrier struct {
	SourceFiles []detectedFile
	Carrier     Carrier
	Errors      []error
}

type inference interface {
	Infer(preliminaryCarrier) preliminaryCarrier
}

var defaultInferences = []inference{
	oneGiantPerformance{},
}

// importInboxCarrier tries to initialise a carrier from a given file path
func (l *Library) importInboxCarrier(filename string) (Carrier, error) {
	carrierID := filename
	if len(carrierID) > 4 && carrierID[len(carrierID)-4:] == ".zip" {
		carrierID = carrierID[:len(carrierID)-4]
	}
	lld := len(l.LibraryDir)
	if len(carrierID) > lld && carrierID[:lld] == l.LibraryDir {
		carrierID = strings.TrimLeft(carrierID[lld:], "/")
	}
	carrierID = strings.ReplaceAll(carrierID, "/", "|")

	rv := preliminaryCarrier{
		Carrier: Carrier{
			ID: carrierID,
		},
	}
	var err error
	rv.SourceFiles, err = listFiles(filename)
	if err != nil {
		rv.Errors = append(rv.Errors, err)
	}

	for _, inf := range defaultInferences {
		rv = inf.Infer(rv)
	}

	// TODO: import inferences from a saved xml file, and apply those

	// Cleanup
	rv = stripSourcePrefix{l.LibraryDir}.Infer(rv)

	return rv.Carrier, multiError(rv.Errors)
}

func listFiles(filename string) ([]detectedFile, error) {
	lfn := len(filename)
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		if lfn > 4 && filename[lfn-4:] == ".zip" {
			return scanZip(filename)
		} else {
			return nil, fmt.Errorf("unknown file type")
		}
	}

	return scanDirectory(filename)
}

func scanDirectory(dirname string) ([]detectedFile, error) {
	var rv []detectedFile

	dir, err := os.Open(dirname)
	if err != nil {
		return rv, err
	}

	fii, err := dir.Readdir(-1)
	if err != nil {
		return rv, err
	}

	var files, dirs []string

	for _, fi := range fii {
		name := fi.Name()
		if name == "" || name[0:1] == "." {
			continue
		}
		fullpath := path.Join(dirname, name)

		if fi.IsDir() {
			dirs = append(dirs, fullpath)
		} else {
			files = append(files, fullpath)
		}
	}

	sort.Strings(files)
	sort.Strings(dirs)

	for _, fullpath := range dirs {
		subf, err := listFiles(fullpath)
		if err != nil {
			return rv, err
		}

		rv = append(rv, subf...)
	}
	for _, fullpath := range files {
		rv = append(rv, detectFile(fullpath))
	}

	return rv, nil
}

func scanZip(archivePath string) ([]detectedFile, error) {
	var rv []detectedFile
	zf, err := zip.OpenReader(archivePath)
	if err != nil {
		return rv, err
	}

	for _, fi := range zf.File {
		rv = append(rv, detectFile(path.Join(archivePath, fi.Name)))
	}

	return rv, nil
}

func multiError(constituentErrors []error) error {
	if len(constituentErrors) == 0 {
		return nil
	} else if len(constituentErrors) == 1 {
		return constituentErrors[0]
	}
	str := "multiple errors:\n"
	for _, err := range constituentErrors {
		str += "\n *  " + strings.ReplaceAll(err.Error(), "\n", "\n    ")
	}
	return fmt.Errorf("%s", str)
}

// refreshInbox reads all carriers yet to be tagged properly, and adds them to the library
func (l *Library) refreshInbox() error {
	rv := []ParsedCarrier{}

	d, err := os.Open(path.Join(l.LibraryDir, "inbox"))
	if err != nil {
		return err
	}

	files, err := d.Readdir(0)
	if err != nil {
		return err
	}
	for _, f := range files {
		fn := f.Name()
		if !f.IsDir() && len(fn) > 4 && fn[len(fn)-4:] == ".xml" {
			// Ignore xml files
			continue
		}

		fileName := path.Join(l.LibraryDir, "inbox", fn)
		carrier, err := l.importInboxCarrier(fileName)

		rv = append(rv, ParsedCarrier{
			Filename: fileName,
			Carrier:  &carrier,
			Error:    err,
		})
	}

	l.Carriers = append(l.Carriers, rv...)
	return nil
}

type oneGiantPerformance struct{}

func (oneGiantPerformance) Infer(pc preliminaryCarrier) preliminaryCarrier {
	carrierID := pc.Carrier.ID
	if len(carrierID) > 7 && (carrierID[:6] == "inbox|" || carrierID[:6] == "inbox/") {
		carrierID = carrierID[6:]
	}
	pf := Performance{
		ID: PerformanceID{pc.Carrier.ID, 2222},
		Work: Work{
			Composer: Composer{},
			Title: []Title{
				{Title: fmt.Sprintf("Auto-detected work in inbox '%s'", carrierID)},
			},
			Year: 2222,
		},
		Year: 2222,
	}

	// Find the longest common prefix of all path-parts
	offset := 0
	ok := true
	var firstFlac detectedFile
	for ok {
		ok = false
		for _, f := range pc.SourceFiles {
			if f.Extension != "flac" {
				continue
			}
			ok = true
			if firstFlac.Path == "" {
				firstFlac = f
			}
			if len(f.Parts) <= offset || f.Parts[offset] != firstFlac.Parts[offset] {
				ok = false
				break
			}
		}
		if ok {
			offset++
		}
	}

	for _, f := range pc.SourceFiles {
		if f.Extension != "flac" {
			continue
		}
		pt := strings.Join(f.Parts[offset:], " - ")
		pf.Work.Parts = append(pf.Work.Parts, Part{
			Part: pt[:len(pt)-5],
		})
		pf.SourceFiles = append(pf.SourceFiles, SourceFile{
			Disc:     f.Disc,
			Filename: f.Path,
		})
	}

	pc.Carrier.Performances = append(pc.Carrier.Performances, pf)
	return pc
}

type stripSourcePrefix struct {
	Prefix string
}

func (s stripSourcePrefix) Infer(pc preliminaryCarrier) preliminaryCarrier {
	lpr := len(s.Prefix)
	for i, pf := range pc.Carrier.Performances {
		for j, sf := range pf.SourceFiles {
			if len(sf.Filename) > lpr && sf.Filename[:lpr] == s.Prefix {
				pc.Carrier.Performances[i].SourceFiles[j].Filename = strings.TrimLeft(sf.Filename[lpr:], "/")
			}
		}
	}
	return pc
}
