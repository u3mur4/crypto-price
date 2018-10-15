package exchange

type Market struct {
	BaseCurrency  string
	QuoteCurrency string
	OpenPrice     float64
	ActualPrice   float64
}

func (m Market) Base() string { return m.BaseCurrency }

func (m Market) Quote() string { return m.QuoteCurrency }

func (m Market) Open() float64 { return m.OpenPrice }

func (m Market) Price() float64 { return m.ActualPrice }
