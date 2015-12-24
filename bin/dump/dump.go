package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/ametheus/speeldoos"
	"io/ioutil"
	"log"
)

var (
	input_file = flag.String("input_file", "", "Input XML file")
)

func init() {
	flag.Parse()
}

func main() {
	ip, err := ioutil.ReadFile(*input_file)
	if err != nil {
		log.Fatal(err)
	}

	foo := &speeldoos.Carrier{}
	err = xml.Unmarshal(ip, foo)
	if err != nil {
		log.Fatal(err)
	}

	js, err := json.MarshalIndent(foo, "", "   ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", js)
}
