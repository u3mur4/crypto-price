package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/u3mur4/crypto-price/exchange"
)

type jsonCandle struct {
	High    float64 `json:"high"`
	Open    float64 `json:"open"`
	Close   float64 `json:"close"`
	Low     float64 `json:"low"`
	Percent float64 `json:"percent"`
	Color   string  `json:"color"`
}

type jsonChart struct {
	Exchange string        `json:"exchange"`
	Base     string        `json:"base"`
	Quote    string        `json:"quote"`
	Candle   jsonCandle    `json:"candle"`
}

type jsonFormat struct {
	Output io.Writer
}

// NewJSON displays the market as json format
func NewJSON() Formatter {
	return &jsonFormat{
		Output: os.Stdout,
	}
}

func (j *jsonFormat) Open() {}

func convertCandles(candle exchange.Candle) (newCandles jsonCandle) {
	return jsonCandle{
		High:    candle.High,
		Open:    candle.Open,
		Close:   candle.Close,
		Low:     candle.Low,
		Percent: candle.Percent(),
		Color:   color(candle).Hex(),
	}
}

func (j *jsonFormat) Show(info exchange.MarketDisplayInfo) {
	market := info.Market

	b, _ := json.Marshal(&jsonChart{
		Exchange: market.Exchange,
		Base:     market.Base,
		Quote:    market.Quote,
		Candle:   convertCandles(market.Candle),
	})

	fmt.Fprintf(j.Output, "%s\n", string(b))
}

func (j *jsonFormat) Close() {}
