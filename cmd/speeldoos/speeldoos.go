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
	Subcommand     string
	ConcurrentJobs int
	LibraryDir     string
	Tools          struct {
		Flac, Metaflac string
		Lame           string
		ID3v2          string
	}
	Extract struct {
		Bitrate string
	}
	Grep struct {
		CaseSensitive bool
	}
	Init struct {
		OutputFile                              string
		TrackFormat, DiscFormat                 string
		Composer                                string
		Year                                    int
		Soloist, Orchestra, Ensemble, Conductor string
		Discs                                   string
	}
	Seedvault struct {
		InputXml, OutputDir                        string
		CoverImage, InlayImage, DiscImage, Booklet string
		EACLogfile, Cuesheet                       string
		NameAfterComposer                          bool
		Tracker                                    string
		DArchive, D320, DV0, DV2, DV6              bool
	}
}{}

var cmdline = flag.NewFlagSet("speeldoos", flag.ContinueOnError)

func init() {
	// Settings {{{
	// Global settings {{{
	cmdline.IntVar(&Config.ConcurrentJobs, "j", 2, "Number of concurrent jobs")
	cmdline.StringVar(&Config.LibraryDir, "library_dir", ".", "Search speeldoos files in this directory")

	// }}}
	// External tools {{{
	cmdline.StringVar(&Config.Tools.Flac, "tools.flac", "", "Path to `flac`")
	cmdline.StringVar(&Config.Tools.Metaflac, "tools.metaflac", "", "Path to `metaflac`")
	cmdline.StringVar(&Config.Tools.Lame, "tools.lame", "", "Path to `lame`")
	cmdline.StringVar(&Config.Tools.ID3v2, "tools.id3v2", "", "Path to `id3v2`")

	// }}}
	// Settings for `sd extract` {{{
	cmdline.StringVar(&Config.Extract.Bitrate, "extract.bitrate", "64k", "Output audio bitrate (mp3)")

	// }}}
	// Settings for `sd grep` {{{
	cmdline.BoolVar(&Config.Grep.CaseSensitive, "grep.I", false, "Perforn case-sensitive matching")

	// }}}
	// Settings for `sd init` {{{
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

	// }}}
	// Settings pertaining to `sd seedvault` {{{
	cmdline.StringVar(&Config.Seedvault.InputXml, "seedvault.input_xml", "", "Input XML file")
	cmdline.StringVar(&Config.Seedvault.OutputDir, "seedvault.output_dir", "seedvault", "Output directory")

	cmdline.StringVar(&Config.Seedvault.CoverImage, "seedvault.cover_image", "", "Path to cover image")
	cmdline.StringVar(&Config.Seedvault.InlayImage, "seedvault.inlay_image", "", "Path to inlay image")
	cmdline.StringVar(&Config.Seedvault.DiscImage, "seedvault.disc_image", "", "Path to disc image")
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

	// }}}
	// }}}

	// Parse config file first, and override with anything on the commandline
	rcfile.ParseInto(cmdline, "speeldoosrc")
	cmdline.Parse(os.Args[1:])

	// HACK: Create aliases for subcommand-specific flags, then call flag.Parse() again.
	args := cmdline.Args()
	if len(args) > 0 {
		Config.Subcommand = args[0]

		if getSubCmd(Config.Subcommand) != nil {
			prefix := Config.Subcommand + "."
			ff := make([]*flag.Flag, 0, 10)
			cmdline.VisitAll(func(f *flag.Flag) {
				if len(f.Name) > len(prefix) && f.Name[0:len(prefix)] == prefix {
					ff = append(ff, f)
				}
			})
			for _, f := range ff {
				cmdline.Var(f.Value, f.Name[len(prefix):], f.Usage)
			}
		}

		cmdline.Parse(args[1:])
	}

	// Sanity checks

	if Config.Tools.Flac == "" {
		Config.Tools.Flac = "flac"
	}
	if Config.Tools.Metaflac == "" {
		Config.Tools.Metaflac = "metaflac"
	}
	if Config.Tools.Lame == "" {
		Config.Tools.Lame = "lame"
	}
	if Config.Tools.ID3v2 == "" {
		Config.Tools.ID3v2 = "id3v2"
	}

	if Config.ConcurrentJobs < 1 {
		Config.ConcurrentJobs = 1
	}
}

type SubCommand func([]string)

func getSubCmd(name string) SubCommand {
	if name == "extract" {
		return extract_main
	} else if name == "grep" {
		return grep_main
	} else if name == "init" {
		return init_main
	} else if name == "seedvault" {
		return seedvault_main
	} else {
		return nil
	}
}

func main() {
	if Config.Subcommand == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s COMMAND [options]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
		return
	}

	cmd := getSubCmd(Config.Subcommand)
	args := cmdline.Args()
	if cmd != nil {
		cmd(args)
	} else {
		fmt.Fprintf(os.Stderr, "Unknown subcommand %s.\n", Config.Subcommand)
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
