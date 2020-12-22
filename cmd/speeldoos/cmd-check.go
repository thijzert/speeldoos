package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/thijzert/speeldoos/lib/ziptraverser"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

type fixableError string

func (f fixableError) Error() string {
	return string(f)
}

func fixErr(format string, a ...interface{}) fixableError {
	return fixableError(fmt.Sprintf(format, a...))
}

type checkF func(*speeldoos.Carrier) []error

var allChecks []checkF = []checkF{
	check_carrierID,
	check_sourceFiles,
	check_composers,
}

func check_main(args []string) {
	allCarr, err := allCarriersWithErrors()
	if err != nil {
		log.Fatalf("Unable to open library: %s", err)
	}

	exitStatus := 0

	for _, pc := range allCarr {
		if pc.Error != nil {
			exitStatus = 1
			log.Printf("Parse error in %s: %s", pc.Filename, pc.Error)
			continue
		}

		modified := false

		for _, f := range allChecks {
			errs := f(pc.Carrier)
			if errs != nil {
				for _, e := range errs {
					if ff, ok := e.(fixableError); ok {
						modified = true
						fmt.Printf("%s: %s (fixed)\n", pc.Filename, ff)
					} else {
						exitStatus = 1
						fmt.Printf("%s: %s\n", pc.Filename, e.Error())
					}
				}
			}
		}

		if modified {
			pc.Carrier.Write(pc.Filename)
		}
	}

	os.Exit(exitStatus)
}

func check_carrierID(foo *speeldoos.Carrier) []error {
	if foo.ID == "" {
		return []error{fmt.Errorf("no carrier ID")}
	}
	return nil
}

func check_sourceFiles(foo *speeldoos.Carrier) []error {
	rv := []error{}

	seen := make([]string, 0)
	zm := ziptraverser.New()
	defer zm.Close()

	for _, perf := range foo.Performances {
		for _, sf := range perf.SourceFiles {
			if !zm.Exists(path.Join(Config.LibraryDir, sf.Filename)) {
				rv = append(rv, fmt.Errorf("source file missing: %s", sf))
			}

			for _, ssf := range seen {
				if sf.Filename == ssf {
					rv = append(rv, fmt.Errorf("duplicate source file: %s", sf))
				}
			}
			seen = append(seen, sf.Filename)
		}
	}

	return rv
}

func check_composers(foo *speeldoos.Carrier) []error {
	rv := []error{}
	for i, perf := range foo.Performances {
		if perf.Work.Composer.ID == "" || perf.Work.Composer.ID == "2222" {
			name := perf.Work.Composer.Name

			if name == "" && perf.Work.Composer.ID == "" {
				continue
			} else if name == "Anonymous" {
				foo.Performances[i].Work.Composer.ID = "Anonymous_work"
			} else {
				foo.Performances[i].Work.Composer.ID = strings.Replace(name, " ", "_", -1)
			}

			rv = append(rv, fixErr("empty composer ID for '%s'", name))
		}
	}

	return rv
}
