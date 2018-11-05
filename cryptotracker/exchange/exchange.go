package exchange

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ExchangeAdapter interface {
	Name() string
	GetOpen(market Market) (Market, error)
	GetLast(market Market) (Market, error)
	Listen(ctx context.Context, markets []Market, updateC chan<- []Market) error
}

type Exchange interface {
	Register(product string) error
	Start(ctx context.Context, updateC chan<- []Market) error
}

type exchange struct {
	adapter ExchangeAdapter
	markets []Market
}

func (e *exchange) Register(product string) error {
	product = strings.ToLower(product)

	pair := strings.Split(product, "-")
	if len(pair) != 2 {
		return fmt.Errorf("invalid product format")
	}

	market := Market{
		ExchangeName:  e.adapter.Name(),
		BaseCurrency:  pair[0],
		QuoteCurrency: pair[1],
	}

	market, err := e.adapter.GetOpen(market)
	if err != nil {
		return err
	}

	market, err = e.adapter.GetLast(market)
	if err != nil {
		return err
	}

	e.markets = append(e.markets, market)
	return nil
}

func (e *exchange) Start(ctx context.Context, updateC chan<- []Market) error {
	if len(e.markets) == 0 {
		return nil
	}

	// refresh open price every hour
	runEvery(ctx, time.Hour, func() {
		for _, market := range e.markets {
			market, err := e.adapter.GetOpen(market)
			if err != nil {
				logrus.WithField("product", market).WithError(err).Warn("cannot get daily open price")
				continue
			}
		}
	})

	// send the initial market state
	updateC <- e.markets

	// start listening to price changes
	err := e.adapter.Listen(ctx, e.markets, updateC)
	if err != nil {
		return err
	}

	return nil
}

func newExchange(adapter ExchangeAdapter) Exchange {
	return &exchange{
		adapter: adapter,
	}
}
