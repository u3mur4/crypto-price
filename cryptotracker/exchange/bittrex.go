package exchange

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/carterjones/signalr"
	"github.com/carterjones/signalr/hubs"
	"github.com/sirupsen/logrus"
)

type bittrex struct {
	socket  *signalr.Client
	markets map[string]*Market
}

type latestTickResult struct {
	Open       float64 `json:"O"`
	High       float64 `json:"H"`
	Low        float64 `json:"L"`
	Close      float64 `json:"C"`
	Volume     float64 `json:"V"`
	TimeStamp  string  `json:"T"`
	BaseVolume float64 `json:"BV"`
}

type latestTickResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message"`
	Result  []latestTickResult `json:"result"`
}

type gettickerResult struct {
	Bid  float64 `json:"Bid"`
	Ask  float64 `json:"Ask"`
	Last float64 `json:"Last"`
}

type gettickerResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Result  gettickerResult `json:"result"`
}

type summaryLiteDeltaMessage struct {
	MarketName string  `json:"M"`
	Last       float64 `json:"l"`
	BaseVolume float64 `json:"m"`
}

type summaryLiteDeltaResponse struct {
	Deltas []summaryLiteDeltaMessage `json:"D"`
}

func (r *summaryLiteDeltaResponse) Get(product string) (summaryLiteDeltaMessage, error) {
	for i := 0; i < len(r.Deltas); i++ {
		if strings.EqualFold(r.Deltas[i].MarketName, product) {
			return r.Deltas[i], nil
		}
	}
	return summaryLiteDeltaMessage{}, fmt.Errorf("not found")
}

// Gets the daily open price using the http api
func (b bittrex) getOpen(product string) (float64, error) {
	respJSON := latestTickResponse{}
	err := httpGetJSON("https://bittrex.com/Api/v2.0/pub/market/GetLatestTick?marketName="+product+"&tickInterval=day", &respJSON)
	if err != nil {
		return 0, err
	}

	if respJSON.Success == false {
		return 0, fmt.Errorf("%v", respJSON.Message)
	}

	return respJSON.Result[0].Open, nil
}

// Gets the actual price using the http api
func (b bittrex) getPrice(product string) (float64, error) {
	respJSON := gettickerResponse{}
	err := httpGetJSON("https://bittrex.com/api/v1.1/public/getticker?market="+product, &respJSON)
	if err != nil {
		return 0, err
	}

	if respJSON.Success == false {
		return 0, fmt.Errorf("%v", respJSON.Message)
	}

	return respJSON.Result.Last, nil
}

func (b *bittrex) register(product string) error {
	product = strings.ToLower(product)

	currencyPair := strings.Split(product, "-")
	if len(currencyPair) != 2 {
		return fmt.Errorf("Invalid market: %s", product)
	}

	open, err := b.getOpen(product)
	if err != nil {
		return err
	}

	price, err := b.getPrice(product)
	if err != nil {
		return err
	}

	b.markets[product] = &Market{
		ExchangeName:  "bittrex",
		BaseCurrency:  currencyPair[1],
		QuoteCurrency: currencyPair[0],
		OpenPrice:     open,
		ActualPrice:   price,
	}

	return nil
}

func (b *bittrex) Listen(products []string) (<-chan Market, <-chan error) {
	for _, product := range products {
		err := b.register(product)
		if err != nil {
			logrus.WithField("product", product).WithError(err).Warn("cannot register bittrex market")
		}
	}

	updateChan := make(chan Market)
	errorChan := make(chan error)
	go b.listen(updateChan, errorChan)
	return updateChan, errorChan
}

func (b *bittrex) listen(updateChan chan<- Market, errorChan chan<- error) {
	defer close(errorChan)
	defer close(updateChan)

	if len(b.markets) == 0 {
		return
	}

	// send the initial market state
	for _, market := range b.markets {
		updateChan <- *market
	}

	// refresh open price
	doneCh := runEvery(time.Hour, func() {
		for product, market := range b.markets {
			open, err := b.getOpen(product)
			if err != nil {
				logrus.WithField("product", product).WithError(err).Warn("cannot get daily open price")
				continue
			}
			market.OpenPrice = open
		}
	})
	defer close(doneCh)

	// receive any price change using websockets
	msgHandler := func(msg signalr.Message) {
		if len(msg.M) > 0 && msg.M[0].M == "uL" {
			resp, err := b.parseSummaryLiteDeltaResponse(msg)
			if err != nil {
				return
			}
			for product, market := range b.markets {
				liteResp, err := resp.Get(product)
				if err != nil {
					continue
				}
				market.ActualPrice = liteResp.Last
				updateChan <- *market
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

	err := b.socket.Run(msgHandler, errHandler)
	if err != nil {
		logrus.WithError(err).Error("cannot start connection")
		errorChan <- err
		return
	}

	if err := b.socket.Send(hubs.ClientMsg{
		H: "c2",
		M: "SubscribeToSummaryLiteDeltas",
		A: []interface{}{},
		I: 0,
	}); err != nil {
		logrus.WithError(err).Error("cannot subscribe to SummaryLiteDeltas")
	}

	err = <-done
	if err != nil {
		logrus.WithError(err).Error("bittrex exchange stopped")
		errorChan <- err
	}
}

func (b bittrex) parseSummaryLiteDeltaResponse(msg signalr.Message) (*summaryLiteDeltaResponse, error) {
	// decode base64
	b64 := msg.M[0].A[0].(string)
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}

	// unpack tha data
	gzipHeader := []byte{
		// magic number
		0x1f, 0x8b,
		// compression method 0x8=deflate
		0x8,
		// FLaGs
		0x00,
		// Modification TIME
		0x0, 0x0, 0x0, 0x0,
		// specific compression methods
		0x4,
		// operating System
		255,
	}
	data = append(gzipHeader, data...)

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	data, _ = ioutil.ReadAll(gr)

	// parse data as json
	respJSON := summaryLiteDeltaResponse{}
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&respJSON)
	if err != nil {
		return nil, err
	}
	return &respJSON, err
}

// NewBittrex tracks a product on bittrex exchange
func NewBittrex() Exchange {
	c := signalr.New(
		"socket.bittrex.com",
		"1.5",
		"/signalr",
		`[{"name":"c2"}]`,
		nil)
	c.Headers["User-Agent"] = "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"

	return &bittrex{
		socket:  c,
		markets: make(map[string]*Market),
	}
}
