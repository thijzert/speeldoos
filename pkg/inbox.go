package pkg

import (
	"fmt"
	"os"
	"path"
	"strings"
)

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

		pc := ParsedCarrier{Filename: path.Join(l.LibraryDir, "inbox", fn)}
		pc.Carrier, pc.Error = l.importInboxCarrier(pc.Filename)

		rv = append(rv, pc)
	}

	l.Carriers = append(l.Carriers, rv...)
	return nil
}

// importInboxCarrier tries to initialise a carrier from a given file path
func (l *Library) importInboxCarrier(filename string) (*Carrier, error) {
	carrierID := filename
	if len(carrierID) > 4 && carrierID[len(carrierID)-4:] == ".zip" {
		carrierID = carrierID[:len(carrierID)-4]
	}
	lld := len(l.LibraryDir)
	if len(carrierID) > lld && carrierID[:lld] == l.LibraryDir {
		carrierID = strings.TrimLeft(carrierID[lld:], "/")
	}

	rv := Carrier{
		ID: carrierID,
	}

	return &rv, fmt.Errorf("not implemented")
}
