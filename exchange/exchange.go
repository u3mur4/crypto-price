package exchange

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type Exchange interface {
	Name() string
	Register(id MarketID) error
	GetMarkets() []Market
	Update(id MarketID, market Market) error
	Start(ctx context.Context, update chan<- Market) error
}

type ExchangeHelper interface {
	Name() string
	GetOpen(id MarketID) (float64, error)
	GetLast(id MarketID) (float64, error)
	Listen(ctx context.Context, update chan<- Market) error
}

type helper struct {
	exchange ExchangeHelper
	markets  []Market
}

func (h *helper) Register(id MarketID) error {
	openPrice, err := h.exchange.GetOpen(id)
	if err != nil {
		return err
	}

	lastPrice, err := h.exchange.GetLast(id)
	if err != nil {
		return err
	}

	market := Market{
		ExchangeName:  h.exchange.Name(),
		BaseCurrency:  id.Base(),
		QuoteCurrency: id.Quote(),
		OpenPrice:     openPrice,
		LastPrice:     lastPrice,
	}
	h.markets = append(h.markets, market)

	return nil
}

func (h *helper) GetMarkets() []Market {
	return h.markets
}

func (h *helper) Update(id MarketID, market Market) error {
	found := false
	for index, market := range h.markets {
		if market.Base() == id.Base() && market.Quote() == id.Quote() {
			h.markets[index] = market
			found = true
		}
	}

	if found == false {
		return os.ErrNotExist
	}
	return nil
}

func (h *helper) Start(ctx context.Context, update chan<- Market) error {
	// refresh open price every hour
	runEvery(ctx, time.Hour, func() {
		for i, market := range h.markets {
			openPrice, err := h.exchange.GetOpen(market)
			if err != nil {
				logrus.WithField("product", h.markets[i]).WithError(err).Warn("cannot get daily open price")
				continue
			}
			h.markets[i].OpenPrice = openPrice
		}
	})

	// send the initial market state
	for _, market := range h.markets {
		update <- market
	}

	// start listening to price changes
	return h.exchange.Listen(ctx, update)
}
