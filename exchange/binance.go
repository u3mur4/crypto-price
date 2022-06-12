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
	exchangeHelper
	client *binancelib.Client
}

func (b binance) getChartTicker(market *Market) string {
	return strings.ToUpper(market.Base + market.Quote)
}

func (b binance) getIntervalString(market *Market) string {
	if market.Interval <= time.Minute {
		return "1m"
	} else if market.Interval <= time.Minute*5 {
		return "5m"
	} else if market.Interval <= time.Minute*15 {
		return "15m"
	} else if market.Interval <= time.Minute*30 {
		return "30m"
	} else if market.Interval <= time.Minute*60 {
		return "1h"
	} else if market.Interval <= time.Hour*4 {
		return "4h"
	} else {
		return "1d"
	}
}

func (b binance) initMarket(market *Market) error {
	kline := b.client.NewKlinesService()
	kline = kline.Symbol(b.getChartTicker(market)).Interval(b.getIntervalString(market)).Limit(1)
	result, err := kline.Do(context.Background())
	if err != nil {
		return err
	}
	market.Candle.High, _ = strconv.ParseFloat(result[0].High, 64)
	market.Candle.Open, _ = strconv.ParseFloat(result[0].Open, 64)
	market.Candle.Close, _ = strconv.ParseFloat(result[0].Close, 64)
	market.Candle.Low, _ = strconv.ParseFloat(result[0].Low, 64)
	return nil
}

func (b *binance) Start(ctx context.Context, update chan<- Market) error {
	for _, market := range b.markets {
		b.initMarket(market)
		update <- *market
		runEvery(context.Background(), market.Interval, func() {
			b.initMarket(market)
		})
	}

	miniHandler := func(event binancelib.WsAllMiniMarketsStatEvent) {
		for _, marketEvent := range event {
			for _, market := range b.markets {
				if strings.EqualFold(b.getChartTicker(market), marketEvent.Symbol) {
					price, err := strconv.ParseFloat(marketEvent.LastPrice, 64)
					if err != nil {
						logrus.WithError(err).WithField("market", marketEvent.Symbol).Error("cannot parse price")
						continue
					}
					market.Candle.Update(price)
					update <- *market
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
	return &binance{
		exchangeHelper: exchangeHelper{
			name: "binance",
		},
		client: binancelib.NewClient("", ""),
	}
}
