package exchange

import (
	"context"
	"time"
)

type fake struct {
}

func (f fake) Name() string {
	return "fake"
}

func (f fake) GetOpen(market Market) (Market, error) {
	market.OpenPrice = 1000
	return market, nil
}

func (f fake) GetLast(market Market) (Market, error) {
	market.ActualPrice = 750
	return market, nil
}

func (f *fake) Listen(ctx context.Context, markets []Market, updateC chan<- []Market) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 100):
			for idx := range markets {
				markets[idx].ActualPrice++
				if markets[idx].ActualPrice/markets[idx].OpenPrice >= 2 {
					markets[idx].ActualPrice = 750
				}
			}
			updateC <- markets
		}
	}
}

func NewFake() Exchange {
	return newExchange(&fake{})
}
