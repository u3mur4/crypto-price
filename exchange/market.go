package exchange

// MarketID identify a specific market
type MarketID interface {
	Base() string
	Quote() string
}

// Market
type Market struct {
	BaseCurrency  string
	QuoteCurrency string
	ExchangeName  string
	OpenPrice     float64
	LastPrice     float64
}

func (m Market) Base() string { return m.BaseCurrency }

func (m Market) Quote() string { return m.QuoteCurrency }

func (m Market) Open() float64 { return m.OpenPrice }

func (m Market) Price() float64 { return m.LastPrice }

func (m Market) Exchange() string { return m.ExchangeName }
