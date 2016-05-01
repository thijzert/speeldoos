package main

import (
	"flag"
	"github.com/thijzert/speeldoos"
	"log"
)

var (
	input_file  = flag.String("input_file", "", "Input XML file")
	output_file = flag.String("output_file", "", "Output XML file")
)

func init() {
	flag.Parse()
}

func main() {
	foo, err := speeldoos.ImportCarrier(*input_file)
	if err != nil {
		log.Fatal(err)
	}

	err = foo.Write(*output_file)
	if err != nil {
		log.Fatal(err)
	}
}
