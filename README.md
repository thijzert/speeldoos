Speeldoos is a database format for Classical Music. The name is Dutch for "music box," which I think is fitting.

[![stability-experimental](https://img.shields.io/badge/stability-experimental-orange.svg)](https://github.com/emersion/stability-badges#experimental)

This project contains the format specification as well as some utilities for working with the database format.

Installation
---------------
Different parts of speeldoos will have different dependencies; install them as needed. In no particular order, the dependencies include:

* FLAC/MetaFLAC
* LAME
* ID3v2

On my machines I installed most of these using one of the following commands:

* `sudo apt-get install flac lame id3v2`
* `sudo pacman -S flac lame id3v2`

However, your mileage may vary.

Currently, this project primarily consists of the format specification (in XSD form) and one command-line utility for maintaining a local speeldoos database.
To install this utility on mac or linux, try running:

    go get -u github.com/thijzert/speeldoos/...

and make sure `$GOPATH/bin` is in your PATH.
Personally, I like to alias `speeldoos` to `sd` (sorry SimpleDefects), which you can do by adding the following lines to your `~/.profile`:

```bash
alias sd="$GOPATH/bin/speeldoos"
```

Afterwards (you may need to restart your shell first), use the `sd` (or `speeldoos` if you prefer) prefix to any sub-script in speeldoos, e.g.:

    sd grep wohltemperirte
    sd init --composer="Johann Sebastian Bach" 48

On Windows, I don't know yet.

Usage
-----
Speeldoos consists of several subcommands, the most important of which are outlined below

### grep
Search your collection using regular expressions.

Usage:

    sd grep PATTERN [PATTERN [...]]

Multiple patterns may be used, but any work matching any of the patterns will be output.

Example

    sd grep bruckner

### extract
Concatenate and transcode each work's parts into large files.

Usage:

    sd extract --input_xml FILE  [OPTIONS]

This command will solve the shufflability problem (see below) as each work is now in exactly one file. Useful for filling your Subsonic library folder.

Example

    sd extract --input_xml /path/to/file.xml --bitrate 192k

### init
Initialize a new empty speeldoos xml file.

Usage:

    sd init [OPTIONS] N [N [...]]

The arguments to the init scripts are the number of parts in each work on this carrier. The output is a speeldoos xml file where all the empty fields have been filled with "2222", which enables one to quickly fill each one using the "Find" / "Find Next" feature in your text editor.

Example

    sd init --composer "Johann Sebastian Bach" 3 4 7 2  > speeldoos.xml
    vim +/2222 speeldoos.xml

### seedvault
Re-tag a ripped cd, create a speeldoos archive file as well as some encodes.

Usage:

    sd seedvault --input_xml speeldoos.xml

This command is best used when you've just ripped a CD or made an online purchase of some sort.
It reads a speeldoos XML file (e.g. one created with `sd init`) and tags and renames the source files (internally) consistently.

By default, it also creates a speeldoos archive, which has all the source files for this particular carrier in one file, as well as an updated speeldoos XML which is aware of the new filenames.

For each of the encodes you enable (choose from: FLAC, MP3 CBR-320, MP3 VBR-V0, VBR-V2, or VBR-V6) `sd seedvault` has the ability to create a private .torrent file of the resulting directory with a tracker URL you specify in order to, um, easily synchronize your new purchase across all your devices.

Example: imagine you've just purchased Bruckner's 7th symphony at Hyperion.

    sd init --composer "Anton Bruckner" > speeldoos.xml
    vim +/2222 speeldoos.xml
    sd seedvault --input_xml speeldoos.xml --output_dir out
    mv out/CDA67916.xml out/CDA67916.zip /path/to/speeldoos-library/

### check
Check your speeldoos library folder for missing information or other errors.

Usage:

    sd check

Some errors can be fixed automatically (such as adding Composer ID's), others will require manual intervention (like providing missing source files).

History
-------
This project was started to scratch a very specific itch, in that every music player (software or otherwise) is absolutely rubbish at classical music.

Most (if not all) software assumes things like "artist" and "album," neither of which appropriately translates into the classical world, which is centered around concepts like "composer," "work," or "performer."
The difference is subtle, but it's very annoying that there's no one true way of representing classical music in existing fields, and different sources will have different conventions: some will correctly use the 'composer' and 'conductor' fields, which will be ignored by most players; others, anticipating this, will abuse the 'artist' field for the composer's name and try to cram as much information as possible in the 'title' field.

So whenever I see an iPod refer to any music as

    Sir Colin Davis - 1. Allegro

I die a little inside.

Also, there's the shufflability problem. When music players use the "shuffle" function, just playing each file in random order is fine as long as your underlying assumption holds that each file is a self-contained four minute unit. 
Most classical music is not.

Works *can* consist of multiple parts, yes, and those usually correspond pretty neatly to tracks on the carrier, be it a CD or regular files.
However, when listening, the individual parts make very little sense out of context (or out of order, even).
Ideally, one would have a shuffle function that does randomize your playlist somewhat, but preserves the integrity of multi-track works.

Being the stubborn arsehole that I am, I set out to create just that.

License
-------
This program and its source code are available under the terms of the BSD 3-clause license.
Find out what that means here: https://www.tldrlegal.com/l/bsd3
