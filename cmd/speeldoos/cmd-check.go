package main

import (
	"log"
	"os"
)

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

		if pc.Carrier.ID == "" {
			exitStatus = 1
			log.Printf("%s: no carrier ID", pc.Filename)
		}
	}

	os.Exit(exitStatus)
}
