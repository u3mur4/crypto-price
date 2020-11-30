package exchange

import "time"

type Candle struct {
	High  float64
	Open  float64
	Close float64
	Low   float64
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
	return Candle{
		High:  m.High * 100000000,
		Open:  m.Open * 100000000,
		Close: m.Close * 100000000,
		Low:   m.Low * 100000000,
	}
}

type Chart struct {
	Exchange string
	Base     string
	Quote    string
	Interval time.Duration
	Candle   Candle
}

func newChart(name, base, quote string, interval time.Duration) *Chart {
	return &Chart{
		Exchange: name,
		Base:     base,
		Quote:    quote,
		Interval: interval,
		Candle:   Candle{},
	}
}
