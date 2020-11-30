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

func (b binance) getChartTicker(chart *Chart) string {
	return strings.ToUpper(chart.Base + chart.Quote)
}

func (b binance) getIntervalString(chart *Chart) string {
	if chart.Interval <= time.Minute {
		return "1m"
	} else if chart.Interval <= time.Minute*5 {
		return "5m"
	} else if chart.Interval <= time.Minute*15 {
		return "15m"
	} else if chart.Interval <= time.Minute*30 {
		return "30m"
	} else if chart.Interval <= time.Minute*60 {
		return "1h"
	} else if chart.Interval <= time.Hour*4 {
		return "4h"
	} else {
		return "1d"
	}
}

func (b binance) initCandle(chart *Chart) error {
	kline := b.client.NewKlinesService()
	kline = kline.Symbol(b.getChartTicker(chart)).Interval(b.getIntervalString(chart)).Limit(1)
	result, err := kline.Do(context.Background())
	if err != nil {
		return err
	}
	chart.Candle.High, _ = strconv.ParseFloat(result[0].High, 64)
	chart.Candle.Open, _ = strconv.ParseFloat(result[0].Open, 64)
	chart.Candle.Close, _ = strconv.ParseFloat(result[0].Close, 64)
	chart.Candle.Low, _ = strconv.ParseFloat(result[0].Low, 64)
	return nil
}

func (b *binance) Start(ctx context.Context, update chan<- Chart) error {
	for _, chart := range b.charts {
		b.initCandle(chart)
		update <- *chart
		runEvery(context.Background(), chart.Interval, func() {
			b.initCandle(chart)
		})
	}

	miniHandler := func(event binancelib.WsAllMiniMarketsStatEvent) {
		for _, marketEvent := range event {
			for _, chart := range b.charts {
				if strings.EqualFold(b.getChartTicker(chart), marketEvent.Symbol) {
					price, err := strconv.ParseFloat(marketEvent.LastPrice, 64)
					if err != nil {
						logrus.WithError(err).WithField("market", marketEvent.Symbol).Error("cannot parse price")
						continue
					}
					chart.Candle.Update(price)
					update <- *chart
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
