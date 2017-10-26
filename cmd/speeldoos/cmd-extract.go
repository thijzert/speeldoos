package main

import (
	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/hivemind"
	"github.com/thijzert/speeldoos/lib/wavreader"
	"log"
)

func extract_main(args []string) {
	if len(args) == 0 {
		log.Fatal("Specify at least one XML file to extract")
	}

	wavconf := wavreader.Config{
		LamePath:   Config.Tools.Lame,
		FlacPath:   Config.Tools.Flac,
		VBRQuality: Config.Condense.Quality,
	}

	hive := hivemind.New(Config.ConcurrentJobs)

	for _, xml := range args {
		foo, err := speeldoos.ImportCarrier(xml)
		croak(err)

		hive.AddJob(condenseJob{wavconf, foo, "."})
	}

	hive.Wait()
}
