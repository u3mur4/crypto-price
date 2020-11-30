package exchange

import (
	"context"
	"math/rand"
	"time"
)

type fake struct {
	exchangeHelper
}

func (f *fake) Start(ctx context.Context, update chan<- Chart) error {
	for _, chart := range f.charts {
		chart.Candle.High = float64(rand.Int31n(1000) + 1000 )
		chart.Candle.Open = float64(rand.Int31n(1000))
		chart.Candle.Low = float64(rand.Int31n(1000))
		chart.Candle.Update(float64(rand.Int31n(1000)))
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 100):
			for _, chart := range f.charts {
				chart.Candle.Update(chart.Candle.Close + 1)
				if chart.Candle.Percent() > 100 {
					chart.Candle.Close = chart.Candle.Open
				}
				update <- *chart
			}
		}
	}
}

// NewFake returns a test exchange
func NewFake() Exchange {
	return &fake{
		exchangeHelper: exchangeHelper{
			name: "fake",
		},
	}
}
