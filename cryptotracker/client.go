package cryptotracker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/sirupsen/logrus"

	"github.com/u3mur4/crypto-price/cryptotracker/exchange"
	"github.com/u3mur4/crypto-price/cryptotracker/format"
)

type Options struct {
	ConvertToSatoshi bool
}

type Client interface {
	Register(format ...string) error
	Start()
	Close() error
	SetFormatter(f format.Formatter)
}

type client struct {
	exchanges map[string]func() exchange.Exchange // exchange name - exchange constructor
	markets   map[string][]string                 // exchange name - markets
	options   Options
	updateC   chan []exchange.Market
	cancel    context.CancelFunc
	formatter format.Formatter
}

// NewClient creates a new default clients
func NewClient(options Options) Client {
	return &client{
		exchanges: map[string]func() exchange.Exchange{
			"coinbase":  exchange.NewCoinbase,
			"bittrex":   exchange.NewBittrex,
			"binance":   exchange.NewBinance,
			"cryptopia": exchange.NewCryptopia,
			"fake":      exchange.NewFake,
		},
		markets:   make(map[string][]string),
		options:   options,
		updateC:   make(chan []exchange.Market, 10),
		formatter: format.NewI3Bar(),
	}
}

func (c *client) SetFormatter(f format.Formatter) {
	c.formatter = f
}

// Register adds a new markets. The format is exchange:marketname
func (c *client) Register(format ...string) error {
	for _, f := range format {
		err := c.register(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) register(format string) error {
	slice := strings.SplitN(format, ":", 2)
	if len(slice) != 2 {
		return fmt.Errorf("invalid format")
	}
	exchangeName := strings.ToLower(slice[0])
	c.markets[exchangeName] = append(c.markets[exchangeName], slice[1])

	return nil
}

func (c *client) applyOptions(m *exchange.Market) {
	if c.options.ConvertToSatoshi && strings.EqualFold(m.Base(), "btc") {
		m.ActualPrice *= 1e8
		m.OpenPrice *= 1e8
	}
}

func (c *client) showMarket(markets []exchange.Market) {
	for _, market := range markets {
		c.applyOptions(&market)
		c.formatter.Show(&market)
	}
}

func (c *client) startExchange(ctx context.Context, name string) error {
	createExchange, ok := c.exchanges[name]
	if !ok {
		return fmt.Errorf("exchange not found")
	}

	exchange := createExchange()
	for _, market := range c.markets[name] {
		err := exchange.Register(market)
		if err != nil {
			return err
		}
	}

	return exchange.Start(ctx, c.updateC)
}

func (c *client) startAllExchange() error {
	c.formatter.Open()
	defer c.formatter.Close()
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.cancel = cancel

	errC := make(chan error)

	for name := range c.markets {
		go func(name string) {
			err := c.startExchange(ctx, name)
			if err != nil {
				errC <- err
			}
		}(name)
	}

	for {
		select {
		case err := <-errC:
			return err
		case markets := <-c.updateC:
			c.showMarket(markets)
		}
	}
}

func (c *client) Start() {
	bf := backoff.NewExponentialBackOff()
	for {
		err := c.startAllExchange()
		if err != nil {
			logrus.WithError(err).Warning("exchanges stoped")
		}

		if bf.GetElapsedTime() >= time.Minute {
			bf.Reset()
		}
		wait := bf.NextBackOff()
		logrus.WithField("duration", wait).Info("wait to restart")
		time.Sleep(wait)
	}
}

func (c *client) Close() error {
	c.cancel()
	return nil
}
