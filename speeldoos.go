package speeldoos

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
