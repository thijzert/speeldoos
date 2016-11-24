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

var cmdline = flag.NewFlagSet("speeldoos", flag.ContinueOnError)

func init() {
	// Global settings
	cmdline.IntVar(&Config.ConcurrentJobs, "j", 2, "Number of concurrent jobs")

	// Settings pertaining to `sd seedvault`
	cmdline.StringVar(&Config.Seedvault.InputXml, "seedvault.input_xml", "", "Input XML file")
	cmdline.StringVar(&Config.Seedvault.OutputDir, "seedvault.output_dir", "seedvault", "Output directory")

	cmdline.StringVar(&Config.Seedvault.CoverImage, "seedvault.cover_image", "", "Path to cover image")
	cmdline.StringVar(&Config.Seedvault.InlayImage, "seedvault.inlay_image", "", "Path to inlay image")
	cmdline.StringVar(&Config.Seedvault.Booklet, "seedvault.booklet", "", "Path to booklet PDF")
	cmdline.StringVar(&Config.Seedvault.EACLogfile, "seedvault.eac_logfile", "", "Path to EAC log file")
	cmdline.StringVar(&Config.Seedvault.Cuesheet, "seedvault.cuesheet", "", "Path to cuesheet")

	cmdline.BoolVar(&Config.Seedvault.NameAfterComposer, "name_after_composer", false, "Name the album after the first composer rather than the first performer")

	cmdline.StringVar(&Config.Seedvault.Tracker, "seedvault.tracker", "", "URL to private tracker")

	cmdline.BoolVar(&Config.Seedvault.DArchive, "seedvault.archive", true, "Create a speeldoos archive")
	cmdline.BoolVar(&Config.Seedvault.D320, "seedvault.320", false, "Also encode MP3-320")
	cmdline.BoolVar(&Config.Seedvault.DV0, "seedvault.v0", false, "Also encode V0")
	cmdline.BoolVar(&Config.Seedvault.DV2, "seedvault.v2", false, "Also encode V2")
	cmdline.BoolVar(&Config.Seedvault.DV6, "seedvault.v6", false, "Also encode V6 (for audiobooks)")

	// Settings for `sd init`
	cmdline.StringVar(&Config.Init.OutputFile, "init.output_file", "", "Output XML file")

	cmdline.StringVar(&Config.Init.TrackFormat, "init.track_format", "track_%02d.flac", "Filename format of the track number")
	cmdline.StringVar(&Config.Init.DiscFormat, "init.disc_format", "disc_%02d", "Directory name format of the disc number")

	cmdline.StringVar(&Config.Init.Composer, "init.composer", "2222", "Preset the composer of each work")
	cmdline.IntVar(&Config.Init.Year, "init.year", 2222, "Preset the year of each performance")

	cmdline.StringVar(&Config.Init.Soloist, "init.soloist", "", "Pre-fill a soloist in each performance")
	cmdline.StringVar(&Config.Init.Orchestra, "init.orchestra", "", "Pre-fill an orchestra in each performance")
	cmdline.StringVar(&Config.Init.Ensemble, "init.ensemble", "", "Pre-fill an ensemble in each performance")
	cmdline.StringVar(&Config.Init.Conductor, "init.conductor", "", "Pre-fill a conductor in each performance")

	cmdline.StringVar(&Config.Init.Discs, "init.discs", "", "A space separated list of the number of tracks in each disc, for a multi-disc release.")

	// Parse config file first, and override with anything on the commandline
	rcfile.ParseInto(cmdline, "speeldoos")
	cmdline.Parse(os.Args[1:])

	// HACK: Create aliases for subcommand-specific flags, then call flag.Parse() again.
	if len(os.Args) > 1 && getSubCmd(os.Args[1]) != nil {
		prefix := os.Args[1] + "."
		ff := make([]*flag.Flag, 0, 10)
		cmdline.VisitAll(func(f *flag.Flag) {
			if len(f.Name) > len(prefix) && f.Name[0:len(prefix)] == prefix {
				ff = append(ff, f)
			}
		})
		for _, f := range ff {
			cmdline.Var(f.Value, f.Name[len(prefix):], f.Usage)
		}
		cmdline.Parse(os.Args[2:])
	}

	// Sanity checks

	if Config.ConcurrentJobs < 1 {
		Config.ConcurrentJobs = 1
	}
}

type SubCommand func([]string)

func getSubCmd(name string) SubCommand {
	if name == "seedvault" {
		return seedvault_main
	} else if name == "init" {
		return init_main
	} else {
		return nil
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s COMMAND [options]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
		return
	}

	cmd := getSubCmd(os.Args[1])
	args := cmdline.Args()
	if cmd != nil {
		cmd(args)
	} else {
		fmt.Fprintf(os.Stderr, "Unknown subcommand %s.\n", os.Args[1])
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
