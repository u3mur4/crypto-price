package format

import "github.com/u3mur4/crypto-price/exchange"

type multiFormat struct {
	formatters []Formatter
}

func NewMulti(f ...Formatter) Formatter {
	return &multiFormat{
		formatters: f,
	}
}

func (m multiFormat) Open() {
	for _, f := range m.formatters {
		f.Open()
	}
}

func (m multiFormat) Show(info exchange.MarketDisplayInfo) {
	for _, formatter := range m.formatters {
		formatter.Show(info)
	}
}

func (m multiFormat) Close() {
	for _, f := range m.formatters {
		f.Close()
	}
}
