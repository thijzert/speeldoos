package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/zipmap"
)

type checkF func(*speeldoos.Carrier) []error

var allChecks []checkF = []checkF{
	check_carrierID,
	check_sourceFiles,
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

		for _, f := range allChecks {
			errs := f(pc.Carrier)
			if errs != nil {
				for _, e := range errs {
					exitStatus = 1
					log.Printf("%s: %s", pc.Filename, e.Error())
				}
			}
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
	zm := zipmap.New()

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
