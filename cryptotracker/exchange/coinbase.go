package exchange

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	ws "github.com/gorilla/websocket"
	gdax "github.com/preichenberger/go-gdax"
	"github.com/sirupsen/logrus"
)

var coinbaseAPI = "https://api.pro.coinbase.com"

type coinbase struct {
	markets map[string]*Market
}

// Gets the daily open price using the http api
func (c coinbase) getOpen(product string) (float64, error) {
	day := time.Now().UTC().Truncate(time.Hour * 24)
	start := day.Add(time.Minute * -5).Format(time.RFC3339)
	end := day.Format(time.RFC3339)
	granularity := 300

	url := fmt.Sprintf("%s/products/%s/candles?start=%s&end=%s&granularity=%d", coinbaseAPI, product, start, end, granularity)
	response := [][]float64{}
	err := httpGetJSON(url, &response)
	if err != nil {
		return 0, err
	}
	// [ time, low, high, open, close, volume ]
	return response[0][4], nil
}

// Gets the actual price using the http api
func (c coinbase) getPrice(product string) (float64, error) {
	url := fmt.Sprintf("%s/products/%s/ticker", coinbaseAPI, product)
	response := struct {
		Price string `json:"price"`
	}{}
	err := httpGetJSON(url, &response)
	if err != nil {
		return 0, err
	}

	p, _ := strconv.ParseFloat(response.Price, 64)
	return p, nil
}

func (c coinbase) register(product string) error {
	product = strings.ToLower(product)

	currencyPair := strings.Split(product, "-")
	if len(currencyPair) != 2 {
		return fmt.Errorf("Invalid market: %s", product)
	}

	open, err := c.getOpen(product)
	if err != nil {
		return err
	}

	price, err := c.getPrice(product)
	if err != nil {
		return err
	}

	c.markets[product] = &Market{
		BaseCurrency:  currencyPair[0],
		QuoteCurrency: currencyPair[1],
		OpenPrice:     open,
		ActualPrice:   price,
	}

	return nil
}

func (c coinbase) Listen(products []string) (<-chan Market, <-chan error) {
	for _, product := range products {
		err := c.register(product)
		if err != nil {
			logrus.WithField("product", product).WithError(err).Warn("cannot register coinbase market")
		}
	}

	updateChan := make(chan Market)
	errorChan := make(chan error)
	go c.listen(updateChan, errorChan)
	return updateChan, errorChan
}

func (c coinbase) listen(updateChan chan<- Market, errorChan chan<- error) {
	defer close(errorChan)
	defer close(updateChan)

	if len(c.markets) == 0 {
		return
	}

	// send the initial market state
	for _, market := range c.markets {
		updateChan <- *market
	}

	// refresh open price
	done := runEvery(time.Hour, func() {
		for product, market := range c.markets {
			open, err := c.getOpen(product)
			if err != nil {
				logrus.WithField("product", product).WithError(err).Warn("cannot get daily open price")
				continue
			}
			market.OpenPrice = open
		}
	})
	defer close(done)

	// receive any price change using websockets
	var wsDialer ws.Dialer
	wsConn, _, err := wsDialer.Dial("wss://ws-feed.gdax.com", nil)
	if err != nil {
		errorChan <- err
		return
	}

	productIDs := []string{}
	for product := range c.markets {
		productIDs = append(productIDs, product)
	}

	subscribe := gdax.Message{
		Type: "subscribe",
		Channels: []gdax.MessageChannel{
			gdax.MessageChannel{
				Name:       "matches",
				ProductIds: productIDs,
			},
		},
	}

	if err := wsConn.WriteJSON(subscribe); err != nil {
		errorChan <- err
		return
	}

	message := gdax.Message{}
	for {
		if err := wsConn.ReadJSON(&message); err != nil {
			errorChan <- err
			return
		}

		if message.Type == "match" {
			if market, ok := c.markets[strings.ToLower(message.ProductId)]; ok {
				market.ActualPrice = message.Price
				updateChan <- *market
			}
		}
	}
}

// NewCoinbase tracks a product on gdax/coinbase exchange
func NewCoinbase() Exchange {
	return coinbase{
		markets: make(map[string]*Market),
	}
}
