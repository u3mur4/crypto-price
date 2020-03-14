package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	binancelib "github.com/adshao/go-binance"
	"github.com/sirupsen/logrus"
)

type binance struct {
	helper
	client *binancelib.Client
}

func (b binance) Name() string {
	return "binance"
}

func (b binance) marketToSymbol(id MarketID) string {
	return strings.ToUpper(id.Base() + id.Quote())
}

func (b binance) GetOpen(id MarketID) (float64, error) {
	resp, err := b.client.NewListPriceChangeStatsService().Symbol(b.marketToSymbol(id)).Do(context.Background())
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(resp[0].OpenPrice, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

// Gets the actual price using the http api
func (b binance) GetLast(id MarketID) (float64, error) {
	resp, err := b.client.NewListPriceChangeStatsService().Symbol(b.marketToSymbol(id)).Do(context.Background())
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(resp[0].LastPrice, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (b *binance) Listen(ctx context.Context, update chan<- Market) error {
	miniHandler := func(event binancelib.WsAllMiniMarketsStatEvent) {
		for _, marketEvent := range event {
			for idx := range b.markets {
				if strings.EqualFold(b.marketToSymbol(b.markets[idx]), marketEvent.Symbol) {
					price, err := strconv.ParseFloat(marketEvent.LastPrice, 64)
					if err != nil {
						logrus.WithError(err).WithField("market", marketEvent.Symbol).Error("cannot parse price")
						continue
					}
					b.markets[idx].LastPrice = price
					update <- b.markets[idx]
				}
			}
		}
	}

	errC := make(chan error)
	errHandler := func(err error) {
		errC <- err
	}

	doneC, stopC, err := binancelib.WsAllMiniMarketsStatServe(miniHandler, errHandler)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		stopC <- struct{}{}
		return ctx.Err()
	case <-doneC:
		return fmt.Errorf("exchange stopped")
	case err := <-errC:
		stopC <- struct{}{}
		return err
	}
}

func NewBinance() Exchange {
	b := &binance{
		client: binancelib.NewClient("", ""),
	}
	b.exchange = b
	return b
}
