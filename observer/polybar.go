package observer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"github.com/u3mur4/crypto-price/exchange"
)

type PolybarConfig struct {
	ShortOnlyOnWeekend bool
}

type PolybarOutput struct {
	markets   map[string]exchange.MarketDisplayInfo
	showPrice map[string]bool
	config    PolybarConfig
	keys      []string
	log       *logrus.Entry
}

func NewPolybarOutput(config PolybarConfig) *PolybarOutput {
	polybar := &PolybarOutput{
		markets:   make(map[string]exchange.MarketDisplayInfo),
		showPrice: make(map[string]bool),
		config:    config,
		keys:      make([]string, 0),
		log:       logrus.WithField("observer", "polybar"),
	}

	go polybar.startConfigServer()

	return polybar
}

func (polybar *PolybarOutput) startConfigServer() {
	process := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				polybar.log.WithError(err).Error("Failed to parse form")
				return
			}

			market := r.FormValue("market")
			if _, ok := polybar.markets[market]; !ok {
				polybar.log.WithField("market", market).Error("Market not found")
				return
			}

			action := r.FormValue("action")
			switch action {
			case "toggle_price":
				if showPrice, ok := polybar.showPrice[market]; ok {
					polybar.showPrice[market] = !showPrice
				} else {
					polybar.showPrice[market] = true
				}
				polybar.log.WithField("market", market).WithField("show", polybar.showPrice[market]).Info("Toggled price visibility")
			default:
				polybar.log.WithField("action", action).Error("Unknown action")
			}
				
			polybar.Update(polybar.markets[market])

		}
	}

	http.HandleFunc("/polybar", process)
	port := ":60253"
	polybar.log.WithField("port", port).Info("Starting config server")
	err := http.ListenAndServe(port, nil)
	if err != nil {
		polybar.log.WithError(err).Error("Config server stopped")
		return
	}
}

func (polybar *PolybarOutput) formatQuote(market exchange.Market) string {
	if strings.EqualFold(market.Quote, "btc") {
		// return "Ƀ"
		return ""
	} else if strings.EqualFold(market.Quote, "usd") || strings.EqualFold(market.Quote, "usdt") {
		return "$"
	} else if strings.EqualFold(market.Quote, "eur") {
		return "€"
	}
	return ""
}

func (polybar *PolybarOutput) formatPrice(market exchange.Market) string {
	if strings.EqualFold(market.Quote, "btc") {
		if market.Candle.Close < 1 {
			return fmt.Sprintf("%.8f", market.Candle.Close)
		}
		return humanize.Comma(int64(market.Candle.Close))
	}
	return fmt.Sprintf("%.0f", market.Candle.Close)
}

func (polybar *PolybarOutput) tooglePrice(market, data string) string {
	b := strings.Builder{}
	b.WriteString("%{A1:")
	b.WriteString("curl -d 'market=" + market + "' -X POST http\\://localhost\\:60253")
	b.WriteString(":}")
	b.WriteString(data)
	b.WriteString("%{A}")
	return b.String()
}

func (polybar *PolybarOutput) Update(info exchange.MarketDisplayInfo) {
	market := info.Market

	key := market.Key()

	// keep output consistent
	if _, ok := polybar.markets[key]; !ok {
		polybar.keys = append(polybar.keys, key)
	}
	polybar.markets[key] = info

	// on weekend only label is visible
	weekDay := time.Now().Weekday()
	if polybar.config.ShortOnlyOnWeekend && (weekDay == time.Saturday || weekDay == time.Sunday) {
		polybar.showPrice[key] = false
	}

	// format all market
	builder := strings.Builder{}
	for _, k := range polybar.keys {
		info := polybar.markets[k]

		price := polybar.formatPrice(info.Market)
		quote := polybar.formatQuote(info.Market)

		builder.WriteString("%{F")
		builder.WriteString(getInterpolatedColorFor(info.Market.Candle).Hex())
		builder.WriteString("}")

		builder.WriteString(polybar.tooglePrice(k, strings.ToUpper(market.Base)))
		if showPrice, ok := polybar.showPrice[k]; !ok || showPrice {
			builder.WriteString(": ")
			builder.WriteString(quote)
			builder.WriteString(price)
			builder.WriteString(fmt.Sprintf(" (%+.1f%%) ", market.Candle.Percent()))
		} else {
			builder.WriteString(" ")
		}

		builder.WriteString("%{F-}")
	}

	builder.WriteString("\n")
	io.Copy(os.Stdout, strings.NewReader(builder.String()))
}
