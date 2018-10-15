package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	binance "github.com/adshao/go-binance"
	"github.com/sirupsen/logrus"
)

type binanceExchange struct {
	client  *binance.Client
	markets map[string]*Market
}

// Gets the daily open price using the http api
func (b binanceExchange) getOpen(product string) (float64, error) {
	day := time.Now().UTC().Truncate(time.Hour * 24)
	start := day.Add(time.Minute*-5).UnixNano() / int64(time.Millisecond)
	resp, err := b.client.NewKlinesService().Symbol(product).StartTime(start).Limit(1).Interval("5m").Do(context.Background())
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(resp[0].Close, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

// Gets the actual price using the http api
func (b binanceExchange) getPrice(product string) (float64, error) {
	resp, err := b.client.NewPriceChangeStatsService().Symbol(product).Do(context.Background())
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(resp.LastPrice, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (b *binanceExchange) register(product string) error {
	currencyPair := strings.Split(product, "-")
	if len(currencyPair) != 2 {
		return fmt.Errorf("Invalid market: %s", product)
	}

	product = strings.ToUpper(currencyPair[0] + currencyPair[1])

	open, err := b.getOpen(product)
	if err != nil {
		return err
	}

	price, err := b.getPrice(product)
	if err != nil {
		return err
	}

	b.markets[product] = &Market{
		BaseCurrency:  currencyPair[0],
		QuoteCurrency: currencyPair[1],
		OpenPrice:     open,
		ActualPrice:   price,
	}

	return nil
}

func (b *binanceExchange) Listen(products []string) (<-chan Market, <-chan error) {
	for _, product := range products {
		err := b.register(product)
		if err != nil {
			logrus.WithField("product", product).WithError(err).Warn("cannot register binance market")
		}
	}

	updateChan := make(chan Market)
	errorChan := make(chan error)
	go b.listen(updateChan, errorChan)
	return updateChan, errorChan
}

func (b *binanceExchange) listen(updateChan chan<- Market, errorChan chan<- error) {
	defer close(errorChan)
	defer close(updateChan)

	if len(b.markets) == 0 {
		return
	}

	// send the initial market state
	for _, market := range b.markets {
		updateChan <- *market
	}

	// refresh open price
	doneCh := runEvery(time.Hour, func() {
		for product, market := range b.markets {
			open, err := b.getOpen(product)
			if err != nil {
				logrus.WithField("product", product).WithError(err).Warn("cannot get daily open price")
				continue
			}
			market.OpenPrice = open
		}
	})
	defer close(doneCh)

	miniHandler := func(event binance.WsAllMiniMarketsStatEvent) {
		for _, marketEvent := range event {
			if _, ok := b.markets[marketEvent.Symbol]; ok {
				price, err := strconv.ParseFloat(marketEvent.LastPrice, 64)
				if err != nil {
					logrus.WithError(err).WithField("market", marketEvent.Symbol).Warn("cannot parse price")
					continue
				}
				b.markets[marketEvent.Symbol].ActualPrice = price
				updateChan <- *b.markets[marketEvent.Symbol]
			}
		}
	}

	errHandler := func(err error) {
		errorChan <- err
	}

	doneC, _, err := binance.WsAllMiniMarketsStatServe(miniHandler, errHandler)
	if err != nil {
		fmt.Println(err)
		return
	}
	<-doneC
}

func NewBinance() Exchange {
	return &binanceExchange{
		client:  binance.NewClient("", ""),
		markets: make(map[string]*Market),
	}
}
