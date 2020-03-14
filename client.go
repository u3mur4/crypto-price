package cryptoprice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/sirupsen/logrus"

	"github.com/u3mur4/crypto-price/exchange"
	"github.com/u3mur4/crypto-price/format"
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
	update    chan exchange.Market
	cancel    context.CancelFunc
	formatter format.Formatter
}

// NewClient creates a new default clients
func NewClient(options Options) Client {
	return &client{
		exchanges: map[string]func() exchange.Exchange{
			"coinbase": exchange.NewCoinbase,
			"bittrex":  exchange.NewBittrex,
			"fake":     exchange.NewFake,
		},
		markets:   make(map[string][]string),
		options:   options,
		update:    make(chan exchange.Market, 1),
		formatter: format.NewI3Bar(format.I3BarConfig{}),
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
	if c.options.ConvertToSatoshi && strings.EqualFold(m.Quote(), "btc") {
		m.LastPrice *= 1e8
		m.OpenPrice *= 1e8
	}
}

func (c *client) showMarket(market exchange.Market) {
	c.applyOptions(&market)
	c.formatter.Show(market)
}

func (c *client) startExchange(ctx context.Context, name string) error {
	createExchange, ok := c.exchanges[name]
	if !ok {
		return fmt.Errorf("exchange not found")
	}

	ex := createExchange()
	for _, marketName := range c.markets[name] {
		marketName = strings.ToLower(marketName)

		pair := strings.Split(marketName, "-")
		if len(pair) != 2 {
			return fmt.Errorf("invalid product format")
		}

		market := exchange.Market{
			ExchangeName:  ex.Name(),
			BaseCurrency:  pair[0],
			QuoteCurrency: pair[1],
		}

		err := ex.Register(market)
		if err != nil {
			return err
		}
	}

	return ex.Start(ctx, c.update)
}

func (c *client) startAllExchange() error {
	c.formatter.Open()
	defer c.formatter.Close()
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.cancel = cancel

	for name := range c.markets {
		go func(name string) {
			bf := backoff.NewExponentialBackOff()
			for {
				err := c.startExchange(ctx, name)
				if err != nil {
					logrus.WithError(err).WithField("name", name).Error("exchange stoped")
				}
				if bf.GetElapsedTime() >= time.Minute {
					bf.Reset()
				}
				wait := bf.NextBackOff()
				logrus.WithField("duration", wait).Info("wait to restart")
				time.Sleep(wait)
			}
		}(name)
	}

	for market := range c.update {
		c.showMarket(market)
	}

	return nil
}

func (c *client) Start() {
	for {
		err := c.startAllExchange()
		if err != nil {
			logrus.WithError(err).Error("exchanges stoped")
		}
	}
}

func (c *client) Close() error {
	c.cancel()
	return nil
}
