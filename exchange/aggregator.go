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
	Show(info MarketDisplayInfo)
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

func (c *Aggregator) AddObservers(formatter ...Formatter) {
	c.formatter = newMulti(formatter...)
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

func (c *Aggregator) applyOptions(info *MarketDisplayInfo) {
	if c.options.ConvertToSatoshi && strings.EqualFold(info.Market.Quote, "btc") {
		info.Market.Candle = info.Market.Candle.ToSatoshi()
	}
}

func (c *Aggregator) showMarket(info MarketDisplayInfo) {
	c.applyOptions(&info)
	c.formatter.Show(info)
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

		err := ex.Register(pair[0], pair[1])
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

	ticker := time.NewTicker(time.Second * 4)
	var info MarketDisplayInfo
	for {
		select {
		case <-ticker.C:
			// we didn't have any market update yet.
			if info.Market.Base == "" {
				continue
			}
			// we don't have to check network connection if we have recently received any data
			if time.Since(info.Market.LastUpdate) <= time.Second*7 {
				info.LastConfirmedConnectionTime = time.Now()
			} else if HasInternetConnection() {
				info.LastConfirmedConnectionTime = time.Now()
			}
			c.showMarket(info)
		case data, ok := <-c.update:
			if !ok {
				logrus.Info("update channel closed")
				break
			}
			info.Market = data
			info.LastConfirmedConnectionTime = time.Now()
			c.showMarket(info)
		}
	}
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
