package exchange

import (
	"context"
	"strconv"
	"strings"

	ws "github.com/gorilla/websocket"
	"github.com/preichenberger/go-coinbasepro/v2"
)

var coinbaseAPI = "https://api.pro.coinbase.com"

type coinbase struct {
	helper
}

func (c coinbase) Name() string {
	return "coinbase"
}

func (c coinbase) marketIDToSymbol(id MarketID) string {
	return id.Base() + "-" + id.Quote()
}

func (c coinbase) GetOpen(id MarketID) (float64, error) {
	client := coinbasepro.NewClient()
	stat, err := client.GetStats(c.marketIDToSymbol(id))
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(stat.Open, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (c coinbase) GetLast(id MarketID) (float64, error) {
	client := coinbasepro.NewClient()
	stat, err := client.GetStats(c.marketIDToSymbol(id))
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(stat.Last, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func (c coinbase) Listen(ctx context.Context, update chan<- Market) error {
	if len(c.markets) == 0 {
		return nil
	}

	// receive any price change using websockets
	var wsDialer ws.Dialer
	wsConn, _, err := wsDialer.Dial("wss://ws-feed.pro.coinbase.com", nil)
	if err != nil {
		return err
	}

	productIDs := []string{}
	for _, market := range c.markets {
		productIDs = append(productIDs, strings.ToUpper(c.marketIDToSymbol(market)))
	}

	subscribe := coinbasepro.Message{
		Type: "subscribe",
		Channels: []coinbasepro.MessageChannel{
			coinbasepro.MessageChannel{
				Name:       "heartbeat",
				ProductIds: productIDs,
			},
			coinbasepro.MessageChannel{
				Name:       "ticker",
				ProductIds: productIDs,
			},
		},
	}

	if err := wsConn.WriteJSON(subscribe); err != nil {
		return err
	}

	messageC := make(chan coinbasepro.Message)
	errC := make(chan error)
	go func() {
		for {
			message := coinbasepro.Message{}
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
			if message.Type == "ticker" {
				for index := range c.markets {
					if strings.EqualFold(c.marketIDToSymbol(c.markets[index]), message.ProductID) {
						price, err := strconv.ParseFloat(message.Price, 64)
						if err != nil {
							continue
						}
						c.markets[index].LastPrice = price
						update <- c.markets[index]
					}
				}
			}
		}
	}
}

// NewCoinbase tracks a product on gdax/coinbase exchange
func NewCoinbase() Exchange {
	c := &coinbase{}
	c.exchange = c
	return c
}
