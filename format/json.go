package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

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
	Interval time.Duration `json:"interval"`
	Candles  jsonCandle    `json:"candles"`
}

type jsonFormat struct {
	Output io.Writer
	first  bool
}

// NewJSON displays the market as json format
func NewJSON() Formatter {
	return &jsonFormat{
		Output: os.Stdout,
		first:  true,
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
	return
}

func (j *jsonFormat) Show(chart exchange.Chart) {
	b, _ := json.Marshal(&jsonChart{
		Exchange: chart.Exchange,
		Base:     chart.Base,
		Quote:    chart.Quote,
		Interval: chart.Interval,
		Candles:  convertCandles(chart.Candle),
	})

	format := ",\n%s"
	if j.first {
		format = "%s"
		j.first = false
	}

	fmt.Fprintf(j.Output, format, string(b))
}

func (j *jsonFormat) Close() {}
