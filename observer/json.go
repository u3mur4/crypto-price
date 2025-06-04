package observer

import (
	"encoding/json"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/u3mur4/crypto-price/exchange"
	"github.com/u3mur4/crypto-price/internal/logger"
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
	Exchange string     `json:"exchange"`
	Base     string     `json:"base"`
	Quote    string     `json:"quote"`
	Candle   jsonCandle `json:"candle"`
}

type JSONOutput struct {
	Output io.Writer
	log    *logrus.Entry
}

// NewJSON displays the market as json format
func NewJSONOutput() *JSONOutput {
	return &JSONOutput{
		Output: os.Stdout,
		log:    logger.Log().WithField("observer", "json_output"),
	}
}

func (j *JSONOutput) toJSONStruct(info exchange.MarketDisplayInfo) jsonChart {
	return jsonChart{
		Exchange: info.Market.Exchange,
		Base:     info.Market.Base,
		Quote:    info.Market.Quote,
		Candle: jsonCandle{
			High:    info.Market.Candle.High,
			Open:    info.Market.Candle.Open,
			Close:   info.Market.Candle.Close,
			Low:     info.Market.Candle.Low,
			Percent: info.Market.Candle.Percent(),
			Color:   getInterpolatedColorFor(info.Market.Candle).Hex(),
		},
	}
}

func (j *JSONOutput) Update(info exchange.MarketDisplayInfo) {
	err := json.NewEncoder(j.Output).Encode(j.toJSONStruct(info))
	if err != nil {
		j.log.WithError(err).Debug("failed to encode market info to json")
	}
}
