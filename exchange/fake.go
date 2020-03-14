package exchange

import (
	"context"
	"time"
)

type fake struct {
	helper
}

func (f fake) Name() string {
	return "fake"
}

func (f fake) GetOpen(id MarketID) (float64, error) {
	return 1000, nil
}

func (f fake) GetLast(id MarketID) (float64, error) {
	return 750, nil
}

func (f *fake) Listen(ctx context.Context, update chan<- Market) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 100):
			for idx := range f.markets {
				f.markets[idx].LastPrice++
				if f.markets[idx].LastPrice/f.markets[idx].OpenPrice >= 2 {
					f.markets[idx].LastPrice = 750
				}
				update <- f.markets[idx]
			}
		}
	}
}

func NewFake() Exchange {
	f := &fake{}
	f.exchange = f
	return f
}
