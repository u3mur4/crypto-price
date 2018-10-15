package cryptotracker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/u3mur4/crypto-price/cryptotracker/exchange"
	"github.com/u3mur4/crypto-price/cryptotracker/format"
)

type Options struct {
	ConvertToSatoshi bool
}

type Client struct {
	exchanges map[string]func() exchange.Exchange // exchange name - exchange constructor
	markets   map[string][]string                 // exchange name - markets
	options   Options
	Formatter format.Formatter
	locker    sync.Locker
	wg        sync.WaitGroup
}

// NewClient creates a new default clients
func NewClient(options Options) Client {
	return Client{
		exchanges: map[string]func() exchange.Exchange{
			"coinbase": exchange.NewCoinbase,
			"bittrex":  exchange.NewBittrex,
			"binance":  exchange.NewBinance,
			"fake":     exchange.NewFake,
		},
		markets:   make(map[string][]string),
		options:   options,
		Formatter: format.NewI3Bar(),
		locker:    &sync.Mutex{},
		wg:        sync.WaitGroup{},
	}
}

// Register adds a new markets. The format is exchange:marketname
func (c *Client) Register(format ...string) error {
	for _, f := range format {
		c.register(f)
	}
	return nil
}

func (c *Client) register(format string) error {
	slice := strings.SplitN(format, ":", 2)
	if len(slice) != 2 {
		return fmt.Errorf("Invalid format: %s", format)
	}
	slice[0] = strings.ToLower(slice[0])
	c.markets[slice[0]] = append(c.markets[slice[0]], slice[1])

	return nil
}

func (c *Client) applyOptions(m *exchange.Market) {
	if c.options.ConvertToSatoshi && strings.EqualFold(m.Quote(), "btc") {
		m.ActualPrice *= 1e8
		m.OpenPrice *= 1e8
	}
}

func (c *Client) showMarket(m exchange.Market) {
	c.locker.Lock()
	defer c.locker.Unlock()
	c.applyOptions(&m)
	c.Formatter.Show(m)
}

func (c *Client) startExchange(name string) error {
	exchange, ok := c.exchanges[name]
	if !ok {
		return fmt.Errorf("exchange not found")
	}

	markets := c.markets[name]
	updateChan, errChan := exchange().Listen(markets)

	running := true
	for running {
		select {
		case m, ok := <-updateChan:
			running = ok
			if ok {
				c.showMarket(m)
			}
		case err, ok := <-errChan:
			running = ok
			if err != nil {
				logrus.WithError(err).WithField("name", name).Error("exchange stopped")
			}
			return err
		}
	}

	return nil
}

func (c *Client) Run() {
	c.Formatter.Open()
	for name := range c.markets {
		c.wg.Add(1)
		go func(name string) {
			defer c.wg.Done()
			for {
				err := c.startExchange(name)
				if err != nil {
					logrus.WithError(err).WithField("name", name).Error("exchange stopped")
				}
			}
		}(name)
	}
	c.wg.Wait()
	c.Formatter.Close()
}
