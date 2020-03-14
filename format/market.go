package format

import colorful "github.com/lucasb-eyer/go-colorful"

// Market represents a currency pair
type Market interface {
	Exchange() string // name of the exchange
	Base() string     // Base currency
	Quote() string    // Quote currency
	Open() float64    // Daily open price
	Price() float64   // Current price
}

func percent(m Market) float64 {
	if m.Open() == 0 {
		return 0
	}
	return (m.Price()/m.Open() - 1) * 100
}

func color(m Market) colorful.Color {
	return defaultColorMap.getInterpolatedColorFor(percent(m))
}
