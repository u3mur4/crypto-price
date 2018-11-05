package exchange

import (
	"fmt"
	"strings"
	"time"

	"github.com/carterjones/signalr"
	"github.com/sirupsen/logrus"
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

// Gets the daily open price using the http api
func (c cryptopia) getOpen(product string) (float64, error) {
	respJSON := getMarketHistory{}

	err := httpGetJSON("https://www.cryptopia.co.nz/api/GetMarketHistory/"+product+"/48", &respJSON)
	if err != nil {
		return 0, err
	}

	if respJSON.Success == false {
		return 0, fmt.Errorf("%s", respJSON.Error)
	}

	day := time.Now().UTC().Truncate(time.Hour * 24).Unix()
	for index, c := range respJSON.Data {
		if c.Timestamp <= day {
			return respJSON.Data[index-1].Price, nil
		}
	}

	return 0, fmt.Errorf("open price not found in market history")
}

func (c cryptopia) getPrice(product string) (float64, error) {
	respJSON := getMarket{}
	err := httpGetJSON("https://www.cryptopia.co.nz/api/GetMarket/"+product, &respJSON)
	if err != nil {
		return 0, err
	}

	if respJSON.Success == false {
		return 0, fmt.Errorf("%v", respJSON.Error)
	}

	return respJSON.Data.Close, nil
}

func (c *cryptopia) register(product string) error {
	product = strings.ToLower(product)

	currencyPair := strings.Split(product, "-")
	if len(currencyPair) != 2 {
		return fmt.Errorf("Invalid market: %s", product)
	}

	// convert market format to cryptopia market format
	product = fmt.Sprintf("%s_%s", currencyPair[1], currencyPair[0])

	open, err := c.getOpen(product)
	if err != nil {
		return err
	}

	price, err := c.getPrice(product)
	if err != nil {
		return err
	}

	c.markets[product] = &Market{
		ExchangeName:  "cryptopia",
		BaseCurrency:  currencyPair[1],
		QuoteCurrency: currencyPair[0],
		OpenPrice:     open,
		ActualPrice:   price,
	}

	return nil
}

func (c *cryptopia) Listen(products []string) (<-chan Market, <-chan error) {
	for _, product := range products {
		err := c.register(product)
		if err != nil {
			logrus.WithField("product", product).WithError(err).Warn("cannot register cryptopia market")
		}
	}

	updateChan := make(chan Market)
	errorChan := make(chan error)
	go c.listen(updateChan, errorChan)
	return updateChan, errorChan
}

func (c *cryptopia) listen(updateChan chan<- Market, errorChan chan<- error) {
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
	doneCh := runEvery(time.Hour, func() {
		for product, market := range c.markets {
			open, err := c.getOpen(product)
			if err != nil {
				logrus.WithField("product", product).WithError(err).Warn("cannot get daily open price")
				continue
			}
			market.OpenPrice = open
		}
	})
	defer close(doneCh)

	msgHandler := func(msg signalr.Message) {
		// for every trade update message
		for _, m := range msg.M {
			if m.M == "SendTradeDataUpdate" {
				for _, a := range m.A {
					// check if the market is watched
					data := a.(map[string]interface{})
					marketName := strings.ToLower(data["Market"].(string))
					if market, ok := c.markets[marketName]; ok {
						market.ActualPrice = data["Last"].(float64)
						updateChan <- *market
					}
				}
			}
		}
	}

	done := make(chan error)
	errHandler := func(err error) {
		if err != nil {
			defer close(done)
			done <- err
		}
	}

	// Start the connection.
	err := c.socket.Run(msgHandler, errHandler)
	if err != nil {
		logrus.WithError(err).Error("cannot start connection")
		errorChan <- err
		return
	}

	err = <-done
	if err != nil {
		logrus.WithError(err).Error("bittrex exchange stopped")
		errorChan <- err
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

	return &cryptopia{
		socket:  c,
		markets: make(map[string]*Market),
	}
}
