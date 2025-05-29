package exchange

import "time"

type Candle struct {
	Open  float64
	High  float64
	Low   float64
	Close float64
}

func (m *Candle) Update(close float64) {
	m.Close = close
	if m.Close > m.High {
		m.High = m.Close
	} else if m.Close < m.Low {
		m.Low = m.Close
	}
}

func (m Candle) Percent() float64 {
	if m.Open == 0 {
		return 0
	}
	return (m.Close/m.Open - 1) * 100
}

func (m Candle) ToSatoshi() Candle {
	const TO_SATOSHI = 100_000_000
	return Candle{
		High:  m.High * TO_SATOSHI,
		Open:  m.Open * TO_SATOSHI,
		Close: m.Close * TO_SATOSHI,
		Low:   m.Low * TO_SATOSHI,
	}
}

type Market struct {
	Exchange   string
	Base       string
	Quote      string
	Candle     Candle
	LastUpdate time.Time
}

type MarketDisplayInfo struct {
	Market Market
	LastConfirmedConnectionTime time.Time
}

func newMarket(name, base, quote string) *Market {
	return &Market{
		Exchange: name,
		Base:     base,
		Quote:    quote,
		Candle:   Candle{},
		LastUpdate: time.Time{},
	}
}
