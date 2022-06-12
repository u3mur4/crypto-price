package exchange

import (
	"context"
	"math/rand"
	"time"
)

type fake struct {
	markets []*Market
}

func (f *fake) Register(base string, quote string) error {
	f.markets = append(f.markets, newMarket("fake", base, quote))
	return nil
}

func (f *fake) Start(ctx context.Context, update chan<- Market) error {
	for _, market := range f.markets {
		market.Candle.High = float64(rand.Int31n(1000) + 1000)
		market.Candle.Open = float64(rand.Int31n(1000))
		market.Candle.Low = float64(rand.Int31n(1000))
		market.Candle.Update(float64(rand.Int31n(1000)))
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 100):
			for _, market := range f.markets {
				market.Candle.Update(market.Candle.Close + 1)
				if market.Candle.Percent() > 100 {
					market.Candle.Close = market.Candle.Open
				}
				update <- *market
			}
		}
	}
}

// NewFake returns a test exchange
func NewFake() Exchange {
	return &fake{}
}
