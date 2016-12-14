package speeldoos

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type Composer struct {
	Name string
	ID   string `xml:"id,attr,omitempty"`
}

type OpusNumber struct {
	Number    string `xml:",chardata"`
	IndexName string `xml:",attr,omitempty"`
}

func (o OpusNumber) String() string {
	if o.Number == "" {
		return ""
	}
	if o.IndexName == "" {
		return fmt.Sprintf("Op. %s", o.Number)
	} else {
		return fmt.Sprintf("%s %s", o.IndexName, o.Number)
	}
}

type Title struct {
	Title    string `xml:",chardata"`
	Language string `xml:",attr,omitempty"`
}

type Work struct {
	Composer   Composer
	Title      []Title
	OpusNumber []OpusNumber
	Parts      []Part `xml:"Parts>Part,omitempty"`
	Year       int
}

type Part struct {
	Part   string `xml:",chardata"`
	Number string `xml:"number,attr,omitempty"`
}

type Performer struct {
	Name string `xml:",chardata"`
	Role string `xml:"role,attr,omitempty"`
}

type SourceFile struct {
	Filename string `xml:",chardata"`
	Disc     int    `xml:"disc,attr,omitempty"`
}

func (s SourceFile) String() string {
	return s.Filename
}

type Performance struct {
	Work        Work
	Year        int          `xml:",omitempty"`
	Performers  []Performer  `xml:"Performers>Performer"`
	SourceFiles []SourceFile `xml:"SourceFiles>File"`
}

type Carrier struct {
	XMLName      xml.Name `xml:"https://www.inurbanus.nl/NS/speeldoos/1.0 Carrier"`
	Name         string
	ID           string        `xml:",omitempty"`
	Hash         string        `xml:"hash,attr,omitempty"`
	Source       string        `xml:"source,attr,omitempty"`
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
	w.Indent("", "	")
	err = w.Encode(c)
	if err != nil {
		return err
	}

	return nil
}
