package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	"github.com/sirupsen/logrus"
)

type binance struct {
	markets []*Market
}

func (b binance) getChartTicker(market *Market) string {
	return strings.ToUpper(market.Base + market.Quote)
}

func (b binance) initMarket(market *Market) error {
	client := binance_connector.NewClient("", "")
	kline := client.NewKlinesService()
	kline = kline.Symbol(b.getChartTicker(market)).Interval("1d").Limit(1)
	result, err := kline.Do(context.Background())
	if err != nil {
		return err
	}
	market.Candle.High, _ = strconv.ParseFloat(result[0].High, 64)
	market.Candle.Open, _ = strconv.ParseFloat(result[0].Open, 64)
	market.Candle.Close, _ = strconv.ParseFloat(result[0].Close, 64)
	market.Candle.Low, _ = strconv.ParseFloat(result[0].Low, 64)
	market.LastUpdate = time.Now()
	return nil
}

func (b *binance) Register(base string, quote string) error {
	b.markets = append(b.markets, newMarket("binance", base, quote))
	return nil
}

func (b *binance) Start(ctx context.Context, update chan<- Market) error {
	for _, market := range b.markets {
		err := b.initMarket(market)
		if err != nil {
			return err
		}
		update <- *market
		runEvery(context.Background(), time.Hour, func() {
			time.Sleep(time.Second * 10)
			b.initMarket(market)
		})
	}

	stream := binance_connector.NewWebsocketStreamClient(true)
	symbolIntervalPair := make(map[string]string)
	for _, market := range b.markets {
		symbolIntervalPair[b.getChartTicker(market)] = "5m"
	}

	handler := func(event *binance_connector.WsKlineEvent) {
		for _, market := range b.markets {
			if strings.EqualFold(b.getChartTicker(market), event.Symbol) {
				price, err := strconv.ParseFloat(event.Kline.Close, 64)
				if err != nil {
					logrus.WithError(err).WithField("market", event.Symbol).Error("cannot parse price")
					continue
				}
				market.Candle.Update(price)
				market.LastUpdate = time.Now()
				update <- *market
			}
		}
	}

	errC := make(chan error)
	errHandler := func(err error) {
		errC <- err
	}

	doneC, stopC, err := stream.WsCombinedKlineServe(symbolIntervalPair, handler, errHandler)
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
		return err
	}
}

func NewBinance() Exchange {
	return &binance{
	}
}
