package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/ametheus/speeldoos"
	"log"
)

var (
	input_xml = flag.String("input_xml", "", "Input XML file")
	input_zip = flag.String("input_zip", "", "Input ZIP file")
)

func init() {
	flag.Parse()
}

func main() {
	foo, err := speeldoos.ImportCarrier(*input_xml)
	if err != nil {
		log.Fatal(err)
	}

	liz := len(*input_zip)
	zf, err := zip.OpenReader(*input_zip)

	for _, pf := range foo.Performances {
		fmt.Printf("Files in '%s' (%s):\n", pf.Work.Title, pf.Work.Composer.Name)

		for i, fn := range pf.SourceFiles {
			fmt.Printf("   %s\n", fn)
			if fn[0:liz] != *input_zip {
				log.Printf("File %d does not appear to be in this zip archive", i)
			} else {
				for j, zfp := range zf.File {
					if zfp.Name == fn[liz+1:] {
						fmt.Printf("     Found at position %d in zip archive!\n", j)
					}
				}
			}
		}
	}
}
