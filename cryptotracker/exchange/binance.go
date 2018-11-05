package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	binancelib "github.com/adshao/go-binance"
	"github.com/sirupsen/logrus"
)

type binance struct {
	client *binancelib.Client
}

func (b binance) Name() string {
	return "binance"
}

func (b binance) marketToSymbol(market Market) string {
	return strings.ToUpper(market.Base() + market.Quote())
}

func (b binance) GetOpen(market Market) (Market, error) {
	day := time.Now().UTC().Truncate(time.Hour * 24)
	start := day.Add(time.Minute*-5).UnixNano() / int64(time.Millisecond)
	resp, err := b.client.NewKlinesService().Symbol(b.marketToSymbol(market)).StartTime(start).Limit(1).Interval("5m").Do(context.Background())
	if err != nil {
		return market, err
	}

	price, err := strconv.ParseFloat(resp[0].Close, 64)
	if err != nil {
		return market, err
	}

	market.OpenPrice = price
	return market, nil
}

// Gets the actual price using the http api
func (b binance) GetLast(market Market) (Market, error) {
	resp, err := b.client.NewPriceChangeStatsService().Symbol(b.marketToSymbol(market)).Do(context.Background())
	if err != nil {
		return market, err
	}

	price, err := strconv.ParseFloat(resp.LastPrice, 64)
	if err != nil {
		return market, err
	}

	market.ActualPrice = price
	return market, nil
}

func (b *binance) Listen(ctx context.Context, markets []Market, updateC chan<- []Market) error {
	miniHandler := func(event binancelib.WsAllMiniMarketsStatEvent) {
		update := false
		for _, marketEvent := range event {
			for idx := range markets {
				if strings.EqualFold(b.marketToSymbol(markets[idx]), marketEvent.Symbol) {
					price, err := strconv.ParseFloat(marketEvent.LastPrice, 64)
					if err != nil {
						logrus.WithError(err).WithField("market", marketEvent.Symbol).Error("cannot parse price")
						continue
					}
					markets[idx].ActualPrice = price
					update = true
				}
			}
		}
		if update {
			updateC <- markets
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
	return newExchange(&binance{
		client: binancelib.NewClient("", ""),
	})
}
