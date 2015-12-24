package speeldoos

import (
	"encoding/xml"
	"io/ioutil"
	"os"
)

type Composer struct {
	Name string
	ID   string `xml:"id,attr"`
}

type OpusNumber struct {
	Number    string `xml:",chardata"`
	IndexName string `xml:",attr,omitempty"`
}

type Title struct {
	Title    string `xml:",chardata"`
	Language string `xml:",attr,omitempty"`
}

type Work struct {
	Composer   Composer
	Title      []Title
	OpusNumber []OpusNumber
	Parts      []string `xml:"Parts>Part,omitempty"`
	Year       int
}

type Performer struct {
	Name string `xml:",chardata"`
	Role string `xml:"role,attr"`
}

type Performance struct {
	Work        Work
	Performers  []Performer `xml:"Performers>Performer"`
	SourceFiles []string    `xml:"SourceFiles>File"`
}

type Carrier struct {
	Name         string
	Hash         string        `xml:"hash,attr"`
	Performances []Performance `xml:"Performances>Performance"`
}

func ImportCarrier(filename string) (*Carrier, error) {
	ip, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	foo := &Carrier{}
	err = xml.Unmarshal(ip, foo)
	if err != nil {
		return nil, err
	}

	return foo, nil
}

func (c *Carrier) Write(filename string) error {
	op, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer op.Close()

	w := xml.NewEncoder(op)
	w.Indent("", "   ")
	err = w.Encode(c)
	if err != nil {
		return err
	}

	return nil
}
