package main

import (
	"flag"
	"fmt"
	"github.com/thijzert/go-rcfile"
	"log"
	"os"
	"path/filepath"
)

var Config = struct {
	ConcurrentJobs int
	Init           struct {
		OutputFile                              string
		TrackFormat, DiscFormat                 string
		Composer                                string
		Year                                    int
		Soloist, Orchestra, Ensemble, Conductor string
		Discs                                   string
	}
	Seedvault struct {
		InputXml, OutputDir             string
		CoverImage, InlayImage, Booklet string
		EACLogfile, Cuesheet            string
		NameAfterComposer               bool
		Tracker                         string
		DArchive, D320, DV0, DV2, DV6   bool
	}
}{}

func init() {
	// Global settings
	flag.IntVar(&Config.ConcurrentJobs, "j", 2, "Number of concurrent jobs")

	// Settings pertaining to `sd seedvault`
	flag.StringVar(&Config.Seedvault.InputXml, "seedvault.input_xml", "", "Input XML file")
	flag.StringVar(&Config.Seedvault.OutputDir, "seedvault.output_dir", "seedvault", "Output directory")

	flag.StringVar(&Config.Seedvault.CoverImage, "seedvault.cover_image", "", "Path to cover image")
	flag.StringVar(&Config.Seedvault.InlayImage, "seedvault.inlay_image", "", "Path to inlay image")
	flag.StringVar(&Config.Seedvault.Booklet, "seedvault.booklet", "", "Path to booklet PDF")
	flag.StringVar(&Config.Seedvault.EACLogfile, "seedvault.eac_logfile", "", "Path to EAC log file")
	flag.StringVar(&Config.Seedvault.Cuesheet, "seedvault.cuesheet", "", "Path to cuesheet")

	flag.BoolVar(&Config.Seedvault.NameAfterComposer, "name_after_composer", false, "Name the album after the first composer rather than the first performer")

	flag.StringVar(&Config.Seedvault.Tracker, "seedvault.tracker", "", "URL to private tracker")

	flag.BoolVar(&Config.Seedvault.DArchive, "seedvault.archive", true, "Create a speeldoos archive")
	flag.BoolVar(&Config.Seedvault.D320, "seedvault.320", false, "Also encode MP3-320")
	flag.BoolVar(&Config.Seedvault.DV0, "seedvault.v0", false, "Also encode V0")
	flag.BoolVar(&Config.Seedvault.DV2, "seedvault.v2", false, "Also encode V2")
	flag.BoolVar(&Config.Seedvault.DV6, "seedvault.v6", false, "Also encode V6 (for audiobooks)")

	// Settings for `sd init`
	flag.StringVar(&Config.Init.OutputFile, "init.output_file", "", "Output XML file")

	flag.StringVar(&Config.Init.TrackFormat, "init.track_format", "track_%02d.flac", "Filename format of the track number")
	flag.StringVar(&Config.Init.DiscFormat, "init.disc_format", "disc_%02d", "Directory name format of the disc number")

	flag.StringVar(&Config.Init.Composer, "init.composer", "2222", "Preset the composer of each work")
	flag.IntVar(&Config.Init.Year, "init.year", 2222, "Preset the year of each performance")

	flag.StringVar(&Config.Init.Soloist, "init.soloist", "", "Pre-fill a soloist in each performance")
	flag.StringVar(&Config.Init.Orchestra, "init.orchestra", "", "Pre-fill an orchestra in each performance")
	flag.StringVar(&Config.Init.Ensemble, "init.ensemble", "", "Pre-fill an ensemble in each performance")
	flag.StringVar(&Config.Init.Conductor, "init.conductor", "", "Pre-fill a conductor in each performance")

	flag.StringVar(&Config.Init.Discs, "init.discs", "", "A space separated list of the number of tracks in each disc, for a multi-disc release.")

	// Parse config file first, and override with anything on the commandline
	rcfile.Parse()
	flag.Parse()

	// Sanity checks

	if Config.ConcurrentJobs < 1 {
		Config.ConcurrentJobs = 1
	}
}

func main() {
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] COMMAND\n", filepath.Base(os.Args[0]))
		os.Exit(1)
		return
	}

	if args[0] == "seedvault" {
		seedvault_main(args[1:])
	} else if args[0] == "init" {
		init_main(args[1:])
	} else {
		fmt.Fprintf(os.Stderr, "Unknown subcommand %s.\n", args[0])
		os.Exit(1)
		return
	}
}

func croak(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func ncroak(n int, e error) int {
	croak(e)
	return n
}
