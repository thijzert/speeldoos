package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/thijzert/speeldoos"
	"log"
)

var (
	input_file = flag.String("input_file", "", "Input XML file")
)

func init() {
	flag.Parse()
}

func main() {
	foo, err := speeldoos.ImportCarrier(*input_file)
	if err != nil {
		log.Fatal(err)
	}

	js, err := json.MarshalIndent(foo, "", "   ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", js)
}
