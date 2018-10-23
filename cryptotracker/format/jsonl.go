package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type jsonLine struct {
	Output io.Writer
}

// NewJSONLine displays the market as json line format
func NewJSONLine() Formatter {
	return jsonLine{
		Output: os.Stdout,
	}
}

func (j jsonLine) Open() {}

func (j jsonLine) Show(m Market) {
	b, _ := json.Marshal(&jsonMarket{
		Exchange: m.Exchange(),
		Base:     m.Base(),
		Quote:    m.Quote(),
		Open:     m.Open(),
		Price:    m.Price(),
		Percent:  percent(m),
		Color:    color(m).Hex(),
	})
	fmt.Fprintf(j.Output, "%s,\n", string(b))
}

func (j jsonLine) Close() {}
