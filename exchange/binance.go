package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	binancelib "github.com/adshao/go-binance/v2"
	"github.com/sirupsen/logrus"
)

type binance struct {
	markets []*Market
	client *binancelib.Client
}

func (b binance) getChartTicker(market *Market) string {
	return strings.ToUpper(market.Base + market.Quote)
}

func (b binance) initMarket(market *Market) error {
	kline := b.client.NewKlinesService()
	kline = kline.Symbol(b.getChartTicker(market)).Interval("1d").Limit(1)
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

func (b *binance) Register(base string, quote string) error {
	b.markets = append(b.markets, newMarket("binance", base, quote))
	return nil
}

func (b *binance) Start(ctx context.Context, update chan<- Market) error {
	for _, market := range b.markets {
		b.initMarket(market)
		update <- *market
		runEvery(context.Background(), time.Hour, func() {
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
		client: binancelib.NewClient("", ""),
	}
}
