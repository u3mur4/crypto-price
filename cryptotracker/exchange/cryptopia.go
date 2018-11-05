package exchange

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/carterjones/signalr"
)

type cryptopia struct {
	socket  *signalr.Client
	markets map[string]*Market
}

type getMarket struct {
	Success bool        `json:"Success"`
	Message interface{} `json:"Message"`
	Data    struct {
		TradePairID    int     `json:"TradePairId"`
		Label          string  `json:"Label"`
		AskPrice       float64 `json:"AskPrice"`
		BidPrice       float64 `json:"BidPrice"`
		Low            float64 `json:"Low"`
		High           float64 `json:"High"`
		Volume         float64 `json:"Volume"`
		LastPrice      float64 `json:"LastPrice"`
		BuyVolume      float64 `json:"BuyVolume"`
		SellVolume     float64 `json:"SellVolume"`
		Change         float64 `json:"Change"`
		Open           float64 `json:"Open"`
		Close          float64 `json:"Close"`
		BaseVolume     float64 `json:"BaseVolume"`
		BuyBaseVolume  float64 `json:"BuyBaseVolume"`
		SellBaseVolume float64 `json:"SellBaseVolume"`
	} `json:"Data"`
	Error interface{} `json:"Error"`
}

type getMarketHistory struct {
	Success bool        `json:"Success"`
	Message interface{} `json:"Message"`
	Data    []struct {
		TradePairID int     `json:"TradePairId"`
		Label       string  `json:"Label"`
		Type        string  `json:"Type"`
		Price       float64 `json:"Price"`
		Amount      float64 `json:"Amount"`
		Total       float64 `json:"Total"`
		Timestamp   int64   `json:"Timestamp"`
	} `json:"Data"`
	Error string `json:"Error"`
}

func (c cryptopia) Name() string {
	return "cryptopia"
}

func (c cryptopia) marketToSymbol(market Market) string {
	return market.Quote() + "_" + market.Base()
}

func (c cryptopia) GetOpen(market Market) (Market, error) {
	respJSON := getMarketHistory{}

	err := httpGetJSON("https://www.cryptopia.co.nz/api/GetMarketHistory/"+c.marketToSymbol(market)+"/48", &respJSON)
	if err != nil {
		return market, err
	}

	if respJSON.Success == false {
		return market, fmt.Errorf("%s", respJSON.Error)
	}

	day := time.Now().UTC().Truncate(time.Hour * 24).Unix()
	for index, c := range respJSON.Data {
		if c.Timestamp <= day {
			market.OpenPrice = respJSON.Data[index-1].Price
			return market, nil
		}
	}

	return market, fmt.Errorf("open price not found in market history")
}

func (c cryptopia) GetLast(market Market) (Market, error) {
	respJSON := getMarket{}
	err := httpGetJSON("https://www.cryptopia.co.nz/api/GetMarket/"+c.marketToSymbol(market), &respJSON)
	if err != nil {
		return market, err
	}

	if respJSON.Success == false {
		return market, fmt.Errorf("%v", respJSON.Error)
	}

	market.ActualPrice = respJSON.Data.Close
	return market, nil
}

func (c *cryptopia) Listen(ctx context.Context, markets []Market, updateC chan<- []Market) error {
	msgHandler := func(msg signalr.Message) {
		// for every trade update message
		update := false

		for _, m := range msg.M {
			if m.M == "SendTradeDataUpdate" {
				for _, a := range m.A {
					// check if the market is watched
					data := a.(map[string]interface{})
					marketName := data["Market"].(string)
					for idx := range markets {
						if strings.EqualFold(marketName, c.marketToSymbol(markets[idx])) {
							markets[idx].ActualPrice = data["Last"].(float64)
							update = true
						}
					}
				}
			}
		}

		if update {
			updateC <- markets
		}
	}

	errC := make(chan error)
	errHandler := func(err error) {
		errC <- err
	}

	// Start the connection.
	err := c.socket.Run(msgHandler, errHandler)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		c.socket.Close()
		return ctx.Err()
	case err := <-errC:
		c.socket.Close()
		return err
	}
}

func NewCryptopia() Exchange {
	c := signalr.New(
		"www.cryptopia.co.nz",
		"1.5",
		"/signalr",
		`[{"name":"notificationhub"}]`,
		nil)
	c.Headers["User-Agent"] = "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"

	return newExchange(&cryptopia{
		socket: c,
	})
}
