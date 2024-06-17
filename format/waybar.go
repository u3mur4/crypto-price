package format

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/u3mur4/crypto-price/exchange"
)

type WaybarConfig struct {
	Sort               string
	Icon               bool
	ShortOnlyOnWeekend bool
}

type WaybarFormat struct {
	markets   map[string]exchange.Market
	showPrice map[string]bool
	config    WaybarConfig
	keys      []string
}

func NewWaybar(config WaybarConfig) Formatter {
	return &WaybarFormat{
		markets:   make(map[string]exchange.Market),
		config:    config,
		showPrice: make(map[string]bool),
	}
}

func (p *WaybarFormat) Open() {
	process := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				fmt.Fprintf(w, "ParseForm() err: %v", err)
				return
			}

			market := r.FormValue("market")
			if showPrice, ok := p.showPrice[market]; ok {
				p.showPrice[market] = !showPrice
				// force to render immediately
				p.Show(p.markets[market])
			}
		}
	}

	http.HandleFunc("/", process)
	go http.ListenAndServe(":60253", nil)
}

func (p *WaybarFormat) formatQuote(market exchange.Market) string {
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

func (i *WaybarFormat) formatPrice(market exchange.Market) string {
	if strings.EqualFold(market.Quote, "btc") {
		if market.Candle.Close < 1 {
			return fmt.Sprintf("%.8f", market.Candle.Close)
		}
		return humanize.Comma(int64(market.Candle.Close))
	}
	return fmt.Sprintf("%.3f", market.Candle.Close)
}

func (i *WaybarFormat) openTradingViewCmd(market exchange.Market) string {
	b := strings.Builder{}

	b.WriteString("chromium --newtab ")
	b.WriteString("https://www.tradingview.com/chart/?symbol=")
	b.WriteString(market.Exchange)
	b.WriteString(":")
	b.WriteString(market.Base)
	b.WriteString(market.Quote)
	b.WriteString(" --profile-directory=\"Profile 2\"")

	return "%{A1:" + strings.Replace(b.String(), ":", "\\:", -1) + ":}"
}

func (i *WaybarFormat) tooglePrice(market, data string) string {
	b := strings.Builder{}
	b.WriteString("%{A1:")
	b.WriteString("curl -d 'market=" + market + "' -X POST http\\://localhost\\:60253")
	b.WriteString(":}")
	b.WriteString(data)
	b.WriteString("%{A}")
	return b.String()
}

func (i *WaybarFormat) Show(market exchange.Market) {
	key := market.Exchange + market.Base + market.Quote

	// keep output consistent
	if _, ok := i.markets[key]; !ok {
		i.keys = append(i.keys, key)
	}
	i.markets[key] = market

	if _, ok := i.showPrice[key]; !ok {
		i.showPrice[key] = true
	}

	// on weekend only label is visible
	weekDay := time.Now().Weekday()
	if i.config.ShortOnlyOnWeekend && (weekDay == time.Saturday || weekDay == time.Sunday) {
		i.showPrice[key] = false
	}

	// format all market
	builder := strings.Builder{}
	for _, k := range i.keys {
		market := i.markets[k]

		price := i.formatPrice(market)
		quote := i.formatQuote(market)

		// {}

		builder.WriteString("<span color='")
		builder.WriteString(color(market.Candle).Hex())
		builder.WriteString("'>")

		builder.WriteString(strings.ToUpper(market.Base))

		if showPrice, ok := i.showPrice[k]; ok && showPrice {
			builder.WriteString(": ")
			builder.WriteString(quote)
			builder.WriteString(price)
			builder.WriteString(fmt.Sprintf(" (%+.1f%%) ", market.Candle.Percent()))
		} else {
			builder.WriteString(" ")
		}

		builder.WriteString("</span>")
	}

	fmt.Fprintln(os.Stdout, builder.String())
	// logrus.Debug(builder.String())
}

func (p WaybarFormat) Close() {
}
