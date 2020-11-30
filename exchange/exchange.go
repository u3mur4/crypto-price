package exchange

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Exchange listens for price changes in realtime
type Exchange interface {
	// Name of the exchange
	Name() string
	// Register a market to listen for price changes
	Register(id MarketID) error
	// Start listening for price changes in the registered markets
	Start(ctx context.Context, update chan<- Market) error
}

type exchangeHelper interface {
	// Name of the exchange
	Name() string
	// Return the open price of the market
	GetOpen(id MarketID) (float64, error)
	// Return the last price of the market
	GetLast(id MarketID) (float64, error)
	// Start listening for price changes in the registered markets
	Listen(ctx context.Context, update chan<- Market) error
}

type helper struct {
	exchange exchangeHelper
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

func (h *helper) Start(ctx context.Context, update chan<- Market) error {
	// refresh open price every *
	runEvery(ctx, time.Minute*15, func() {
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
