package observer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/sirupsen/logrus"
	"github.com/u3mur4/crypto-price/exchange"
)

type WaybarConfig struct {
	ShortOnlyOnWeekend bool
}

type WaybarOutput struct {
	markets   map[string]exchange.MarketDisplayInfo
	showPrice map[string]bool
	showColor map[string]bool
	config    WaybarConfig
	keys      []string
	log       *logrus.Entry
}

func NewWaybarOutput(config WaybarConfig) *WaybarOutput {
	waybar := &WaybarOutput{
		markets:   make(map[string]exchange.MarketDisplayInfo),
		showPrice: make(map[string]bool),
		showColor: make(map[string]bool),
		config:    config,
		keys:      make([]string, 0),
		log:       logrus.WithField("observer", "waybar"),
	}

	go waybar.startConfigServer()

	return waybar
}

func (waybar *WaybarOutput) startConfigServer() {
	process := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				waybar.log.WithError(err).Error("Failed to parse form")
				return
			}

			market := r.FormValue("market")
			if _, ok := waybar.markets[market]; !ok {
				waybar.log.WithField("market", market).Error("Market not found")
				return
			}

			switch r.URL.Path {
			case "/toggle_price":
				if showPrice, ok := waybar.showPrice[market]; ok {
					waybar.log.WithField("market", market).WithField("show", !showPrice).Info("Toggled price visibility")
					waybar.showPrice[market] = !showPrice
				}
			case "/toggle_color":
				if showColor, ok := waybar.showColor[market]; ok {
					waybar.log.WithField("market", market).WithField("show", !showColor).Info("Toggled color visibility")
					waybar.showColor[market] = !showColor
				}
			}

			// force to render immediately
			waybar.Update(waybar.markets[market])
		}
	}

	http.HandleFunc("/waybar", process)
	port := ":60254"
	waybar.log.WithField("port", port).Info("Starting config server")
	err := http.ListenAndServe(port, nil)
	if err != nil {
		waybar.log.WithError(err).Error("Config server stopped")
	}
}

func (waybar *WaybarOutput) formatQuote(market exchange.Market) string {
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

func (waybar *WaybarOutput) formatPrice(market exchange.Market) string {
	if strings.EqualFold(market.Quote, "btc") {
		if market.Candle.Close < 1 {
			return fmt.Sprintf("%.8f", market.Candle.Close)
		}
		return humanize.Comma(int64(market.Candle.Close))
	}
	return fmt.Sprintf("%.3f", market.Candle.Close)
}

func (waybar *WaybarOutput) Update(info exchange.MarketDisplayInfo) {
	market := info.Market
	key := market.Key()

	// keep output consistent
	if _, ok := waybar.markets[key]; !ok {
		waybar.keys = append(waybar.keys, key)
	}
	waybar.markets[key] = info

	// on weekend only label is visible
	weekDay := time.Now().Weekday()
	if waybar.config.ShortOnlyOnWeekend && (weekDay == time.Saturday || weekDay == time.Sunday) {
		waybar.showPrice[key] = false
	}

	colorWithNetworkConnectionStatus := func(info exchange.MarketDisplayInfo) colorful.Color {
		if time.Since(info.LastConfirmedConnectionTime) > time.Second*5 || time.Since(info.Market.LastUpdate) > time.Second*30 {
			return colorful.Color{R: 0.5, G: 0.5, B: 0.5} // gray
		}
		return getInterpolatedColorFor(info.Market.Candle)
	}

	// format all market
	builder := strings.Builder{}
	for _, k := range waybar.keys {
		info := waybar.markets[k]
		market := info.Market

		price := waybar.formatPrice(market)
		quote := waybar.formatQuote(market)

		builder.WriteString("<span color='")
		if showColor, ok := waybar.showColor[k]; !ok || showColor {
			builder.WriteString(colorWithNetworkConnectionStatus(info).Hex())
		} else {
			builder.WriteString("#FFFFFF")
		}
		builder.WriteString("'>")

		builder.WriteString(strings.ToUpper(market.Base))

		if showPrice, ok := waybar.showPrice[k]; !ok || showPrice {
			builder.WriteString(": ")
			builder.WriteString(quote)
			builder.WriteString(price)
			builder.WriteString(fmt.Sprintf(" (%+.1f%%) ", market.Candle.Percent()))
		} else {
			builder.WriteString(" ")
		}

		builder.WriteString("</span>")
	}

	builder.WriteString("\n")
	io.Copy(os.Stdout, strings.NewReader(builder.String()))
}
