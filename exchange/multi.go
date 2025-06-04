package exchange

type multiFormat struct {
	formatters []Formatter
}

func newMulti(f ...Formatter) Formatter {
	return &multiFormat{
		formatters: f,
	}
}

func (m multiFormat) Open() {
	for _, f := range m.formatters {
		f.Open()
	}
}

func (m multiFormat) Show(info MarketDisplayInfo) {
	for _, formatter := range m.formatters {
		formatter.Show(info)
	}
}

func (m multiFormat) Close() {
	for _, f := range m.formatters {
		f.Close()
	}
}
