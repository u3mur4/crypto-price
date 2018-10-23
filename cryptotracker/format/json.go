package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type jsonFormat struct {
	Output io.Writer
	first  bool
}

type jsonMarket struct {
	Exchange string  `json:"exchange"`
	Base     string  `json:"base"`
	Quote    string  `json:"quote"`
	Open     float64 `json:"open"`
	Price    float64 `json:"price"`
	Percent  float64 `json:"percent"`
	Color    string  `json:"color"`
}

// NewJSON displays the market as json format
func NewJSON() Formatter {
	return &jsonFormat{
		Output: os.Stdout,
		first:  true,
	}
}

func (j *jsonFormat) Open() { fmt.Fprintf(j.Output, "[") }

func (j *jsonFormat) Show(m Market) {
	b, _ := json.Marshal(&jsonMarket{
		Exchange: m.Exchange(),
		Base:     m.Base(),
		Quote:    m.Quote(),
		Open:     m.Open(),
		Price:    m.Price(),
		Percent:  percent(m),
		Color:    color(m).Hex(),
	})

	format := ",\n%s"
	if j.first {
		format = "%s"
		j.first = false
	}

	fmt.Fprintf(j.Output, format, string(b))
}

func (j *jsonFormat) Close() { fmt.Fprintf(j.Output, "]\n") }
