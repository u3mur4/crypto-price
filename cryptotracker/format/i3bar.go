package format

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type i3BarFormat struct {
	Output  io.Writer
	markets map[string]Market
	keys    []string
	printer *message.Printer
}

// NewI3Bar displays the market as i3bar format
func NewI3Bar() Formatter {
	return &i3BarFormat{
		Output:  os.Stdout,
		markets: make(map[string]Market),
		printer: message.NewPrinter(language.English),
	}
}

func (i i3BarFormat) Open() {}

func (i *i3BarFormat) Show(m Market) {
	key := m.Base() + m.Quote()

	// keep output consistent
	if _, ok := i.markets[key]; !ok {
		i.keys = append(i.keys, key)
	}
	i.markets[key] = m

	for _, k := range i.keys {
		m := i.markets[k]
		prec := -1
		if m.Price() < 0.1 {
			prec = 8
		}
		priceFormat := strconv.FormatFloat(m.Price(), 'f', prec, 32)
		i.printer.Fprintf(i.Output, "<span foreground='%s'>%s: "+priceFormat+" (%+.1f%%)</span> ", color(m).Hex(), strings.ToUpper(m.Base()), percent(m))
	}

	if len(i.markets) > 0 {
		fmt.Fprintln(i.Output)
	}
}

func (i i3BarFormat) Close() {
}
