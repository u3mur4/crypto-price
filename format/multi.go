package format

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

func (m multiFormat) Show(market Market) {
	for _, f := range m.formatters {
		f.Show(market)
	}
}

func (m multiFormat) Close() {
	for _, f := range m.formatters {
		f.Close()
	}
}
