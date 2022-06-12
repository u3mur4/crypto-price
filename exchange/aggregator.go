package exchange

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

type Formatter interface {
	Open()
	Show(market Market)
	Close()
}

type Options struct {
	ConvertToSatoshi bool
}

type Aggregator struct {
	exchanges map[string]func() Exchange // exchange name - exchange constructor
	markets   map[string][]string        // exchange name - markets
	options   Options
	update    chan Market
	cancel    context.CancelFunc
	formatter Formatter
}

// NewAggregator creates a new default clients
func NewAggregator(options Options, formatter Formatter) *Aggregator {
	return &Aggregator{
		exchanges: map[string]func() Exchange{
			"binance": NewBinance,
			"fake":    NewFake,
		},
		markets:   make(map[string][]string),
		options:   options,
		update:    make(chan Market, 1),
		formatter: formatter,
	}
}

func (c *Aggregator) SetFormatter(formatter Formatter) {
	c.formatter = formatter
}

// Register adds a new markets. The format is exchange:marketname
func (c *Aggregator) Register(format ...string) error {
	for _, f := range format {
		err := c.register(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Aggregator) register(format string) error {
	slice := strings.SplitN(format, ":", 2)
	if len(slice) != 2 {
		return fmt.Errorf("invalid format")
	}
	exchangeName := strings.ToLower(slice[0])
	c.markets[exchangeName] = append(c.markets[exchangeName], slice[1])

	return nil
}

func (c *Aggregator) applyOptions(market *Market) {
	if c.options.ConvertToSatoshi && strings.EqualFold(market.Quote, "btc") {
		market.Candle = market.Candle.ToSatoshi()
	}
}

func (c *Aggregator) showMarket(market Market) {
	c.applyOptions(&market)
	c.formatter.Show(market)
}

func (c *Aggregator) startExchange(ctx context.Context, name string) error {
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

		err := ex.Register(pair[0], pair[1], time.Hour*24)
		if err != nil {
			return err
		}
	}

	return ex.Start(ctx, c.update)
}

func (c *Aggregator) startAllExchange() error {
	c.formatter.Open()
	defer c.formatter.Close()
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

func (c *Aggregator) Start() {
	for {
		err := c.startAllExchange()
		defer c.cancel()
		if err != nil {
			logrus.WithError(err).Error("exchanges stoped")
		}
	}
}

func (c *Aggregator) Close() error {
	c.cancel()
	return nil
}
