package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	ws "github.com/gorilla/websocket"
	gdax "github.com/preichenberger/go-gdax"
)

var coinbaseAPI = "https://api.pro.coinbase.com"

type coinbase struct {
}

func (c coinbase) Name() string {
	return "coinbase"
}

func (c coinbase) marketToSymbol(market Market) string {
	return market.Base() + "-" + market.Quote()
}

// Gets the daily open price using the http api
func (c coinbase) GetOpen(market Market) (Market, error) {
	day := time.Now().UTC().Truncate(time.Hour * 24)
	start := day.Add(time.Minute * -5).Format(time.RFC3339)
	end := day.Format(time.RFC3339)
	granularity := 300

	url := fmt.Sprintf("%s/products/%s/candles?start=%s&end=%s&granularity=%d", coinbaseAPI, c.marketToSymbol(market), start, end, granularity)
	response := [][]float64{}
	err := httpGetJSON(url, &response)
	if err != nil {
		return market, err
	}
	// [ time, low, high, open, close, volume ]
	market.OpenPrice = response[0][4]
	return market, nil
}

func (c coinbase) GetLast(market Market) (Market, error) {
	url := fmt.Sprintf("%s/products/%s/ticker", coinbaseAPI, c.marketToSymbol(market))
	response := struct {
		Price string `json:"price"`
	}{}
	err := httpGetJSON(url, &response)
	if err != nil {
		return market, err
	}

	price, err := strconv.ParseFloat(response.Price, 64)
	if err != nil {
		return market, err
	}

	market.ActualPrice = price
	return market, nil
}

func (c coinbase) Listen(ctx context.Context, markets []Market, updateC chan<- []Market) error {
	// receive any price change using websockets
	var wsDialer ws.Dialer
	wsConn, _, err := wsDialer.Dial("wss://ws-feed.gdax.com", nil)
	if err != nil {
		return err
	}

	productIDs := []string{}
	for _, market := range markets {
		productIDs = append(productIDs, c.marketToSymbol(market))
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
		return err
	}

	messageC := make(chan gdax.Message)
	errC := make(chan error)
	go func() {
		for {
			message := gdax.Message{}
			if err := wsConn.ReadJSON(&message); err != nil {
				errC <- err
				return
			}
			messageC <- message
		}
	}()

	for {
		select {
		case <-ctx.Done():
			wsConn.Close()
			return ctx.Err()
		case err := <-errC:
			wsConn.Close()
			return err
		case message := <-messageC:
			if message.Type == "match" {
				for idx := range markets {
					if strings.EqualFold(c.marketToSymbol(markets[idx]), message.ProductId) {
						markets[idx].ActualPrice = message.Price
						updateC <- markets
					}
				}
			}
		}
	}
}

// NewCoinbase tracks a product on gdax/coinbase exchange
func NewCoinbase() Exchange {
	return newExchange(coinbase{})
}
