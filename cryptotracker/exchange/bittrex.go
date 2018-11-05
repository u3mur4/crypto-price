package exchange

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/carterjones/signalr"
	"github.com/carterjones/signalr/hubs"
	"github.com/sirupsen/logrus"
)

type bittrex struct {
	socket *signalr.Client
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

func (b bittrex) Name() string {
	return "bittrex"
}

func (b bittrex) marketToSymbol(market Market) string {
	return market.Base() + "-" + market.Quote()
}

func (b bittrex) GetOpen(market Market) (Market, error) {
	respJSON := latestTickResponse{}
	err := httpGetJSON("https://bittrex.com/Api/v2.0/pub/market/GetLatestTick?marketName="+b.marketToSymbol(market)+"&tickInterval=day", &respJSON)
	if err != nil {
		return market, err
	}

	if respJSON.Success == false {
		return market, fmt.Errorf("%v", respJSON.Message)
	}

	market.OpenPrice = respJSON.Result[0].Open
	return market, nil
}

func (b bittrex) GetLast(market Market) (Market, error) {
	respJSON := gettickerResponse{}
	err := httpGetJSON("https://bittrex.com/api/v1.1/public/getticker?market="+b.marketToSymbol(market), &respJSON)
	if err != nil {
		return market, err
	}

	if respJSON.Success == false {
		return market, fmt.Errorf("%v", respJSON.Message)
	}

	market.ActualPrice = respJSON.Result.Last
	return market, nil
}

func (b *bittrex) Listen(ctx context.Context, markets []Market, updateC chan<- []Market) error {
	// receive any price change using websockets
	msgHandler := func(msg signalr.Message) {
		if len(msg.M) > 0 && msg.M[0].M == "uL" {
			resp, err := b.parseSummaryLiteDeltaResponse(msg)
			if err != nil {
				return
			}
			update := false
			for _, market := range markets {
				liteResp, err := resp.Get(b.marketToSymbol(market))
				if err != nil {
					continue
				}
				market.ActualPrice = liteResp.Last
				update = true
			}
			if update {
				updateC <- markets
			}
		}
	}

	errC := make(chan error)
	errHandler := func(err error) {
		errC <- err
	}

	err := b.socket.Run(msgHandler, errHandler)
	if err != nil {
		return err
	}

	if err := b.socket.Send(hubs.ClientMsg{
		H: "c2",
		M: "SubscribeToSummaryLiteDeltas",
		A: []interface{}{},
		I: 0,
	}); err != nil {
		logrus.WithError(err).Error("cannot subscribe to SummaryLiteDeltas")
	}

	select {
	case <-ctx.Done():
		b.socket.Close()
		return ctx.Err()
	case err := <-errC:
		b.socket.Close()
		return err
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

	return newExchange(&bittrex{
		socket: c,
	})
}
