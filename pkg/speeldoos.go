package pkg

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

// A Composer represents a composer
type Composer struct {
	Name string

	// A composer's ID is the page ID for their Wikipedia page
	ID string `xml:"id,attr,omitempty"`
}

// An OpusNumber represents an opus number for a composition.
// Opus numbers, together with the composer, ideally uniquely identify a work
type OpusNumber struct {
	Number string `xml:",chardata"`

	// Some composers use a different index (e.g. 'BWV', 'KV', 'Wq', etc.)
	IndexName string `xml:",attr,omitempty"`
}

func (o OpusNumber) String() string {
	if o.Number == "" {
		return ""
	}
	if o.IndexName == "" {
		return fmt.Sprintf("Op. %s", o.Number)
	}

	return fmt.Sprintf("%s %s", o.IndexName, o.Number)
}

// A Title represents a work's title in a certain language
type Title struct {
	Title    string `xml:",chardata"`
	Language string `xml:",attr,omitempty"`
}

// A Work represents a musical composition.
type Work struct {
	Composer Composer

	// A work may have more than one title in different languages.
	Title []Title

	// The opus number(s) for this composition. There may be more than one index
	// for identifying works by this particular composer. Or none at all. Or
	// this work may just not appear on any of them.
	OpusNumber []OpusNumber

	// The parts that comprise this work, if any.
	Parts []Part `xml:"Parts>Part,omitempty"`

	// The year this composition was completed.
	Year int
}

// A Part of a work
type Part struct {
	Part   string `xml:",chardata"`
	Number string `xml:"number,attr,omitempty"`
}

// A Performer in one particular recording of a Work.
// An 'artist', if you will.
type Performer struct {
	Name string `xml:",chardata"`

	// The type of performer. Suggested values:
	//   * "performer"
	//   * "soloist"
	//   * "orchestra"
	//   * "ensemble"
	//   * "conductor"
	Role string `xml:"role,attr,omitempty"`
}

// A SourceFile wraps the path to a file containing the recording of that particular performance
type SourceFile struct {
	Filename string `xml:",chardata"`

	// The disc number, if applicable.
	Disc int `xml:"disc,attr,omitempty"`
}

func (s SourceFile) String() string {
	return s.Filename
}

// A Performance represents one (recording of a) performance of a Work.
type Performance struct {
	Work Work

	// The year in which the performance took place
	Year int `xml:",omitempty"`

	Performers  []Performer  `xml:"Performers>Performer"`
	SourceFiles []SourceFile `xml:"SourceFiles>File"`
}

// A Carrier can represent either a physical medium or a packaged download.
// An "album," in layman's terms.
type Carrier struct {
	XMLName xml.Name `xml:"https://www.inurbanus.nl/NS/speeldoos/1.0 Carrier"`

	Name string

	// A catalog number, or some other unique identifier within the collection
	ID string `xml:",omitempty"`

	// The SHA256 hash of the corresponding .zip bundle for this carrier.
	Hash string `xml:"hash,attr,omitempty"`

	// An indication of how this carrier ended up in this library.
	// Suggested values:
	//   * "WEB" (digital download)
	//   * "CD" (ripped cd)
	Source string `xml:"source,attr,omitempty"`

	// The performances on this carrier
	Performances []Performance `xml:"Performances>Performance"`
}

// ImportCarrier reads a serialized Carrier from a file
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

// Write serialises the carrier to disk
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
