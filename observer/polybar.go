package observer

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/u3mur4/crypto-price/exchange"
)

type PolybarConfig struct {
	Sort               string
	Icon               bool
	ShortOnlyOnWeekend bool
}

type polybarFormat struct {
	markets   map[string]exchange.MarketDisplayInfo
	showPrice map[string]bool
	config    PolybarConfig
	keys      []string
}

func NewPolybar(config PolybarConfig) Formatter {
	return &polybarFormat{
		markets:   make(map[string]exchange.MarketDisplayInfo),
		config:    config,
		showPrice: make(map[string]bool),
	}
}

func (p *polybarFormat) Open() {
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

func (p *polybarFormat) formatQuote(market exchange.Market) string {
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

func (i *polybarFormat) formatPrice(market exchange.Market) string {
	if strings.EqualFold(market.Quote, "btc") {
		if market.Candle.Close < 1 {
			return fmt.Sprintf("%.8f", market.Candle.Close)
		}
		return humanize.Comma(int64(market.Candle.Close))
	}
	return fmt.Sprintf("%.0f", market.Candle.Close)
}

func (i *polybarFormat) openTradingViewCmd(market exchange.Market) string {
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

func (i *polybarFormat) tooglePrice(market, data string) string {
	b := strings.Builder{}
	b.WriteString("%{A1:")
	b.WriteString("curl -d 'market=" + market + "' -X POST http\\://localhost\\:60253")
	b.WriteString(":}")
	b.WriteString(data)
	b.WriteString("%{A}")
	return b.String()
}

func (i *polybarFormat) Show(info exchange.MarketDisplayInfo) {
	market := info.Market

	key := market.Exchange + market.Base + market.Quote

	// keep output consistent
	if _, ok := i.markets[key]; !ok {
		i.keys = append(i.keys, key)
	}
	i.markets[key] = info

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
		info := i.markets[k]

		price := i.formatPrice(info.Market)
		quote := i.formatQuote(info.Market)
		// icon := i.getIcon(market)

		// // use icon or the base
		// 	tmp := fmt.Sprintf("<span foreground='%s'>%s: </span>", color(market).Hex(), strings.ToUpper(market.Base()))
		// 	builder.WriteString(tmp)
		// }

		// // print price
		// tmp := fmt.Sprintf("<span foreground='%s'>%s%s (%+.1f%%)</span> ", color(market).Hex(), price, quote, percent(market))
		// builder.WriteString(tmp)

		builder.WriteString("%{F")
		builder.WriteString(color(info.Market.Candle).Hex())
		builder.WriteString("}")

		// builder.WriteString(i.openTradingViewCmd(chart))

		builder.WriteString(i.tooglePrice(k, strings.ToUpper(market.Base)))
		if showPrice, ok := i.showPrice[k]; ok && showPrice {
			builder.WriteString(": ")
			builder.WriteString(quote)
			builder.WriteString(price)
			builder.WriteString(fmt.Sprintf(" (%+.1f%%) ", market.Candle.Percent()))
		} else {
			builder.WriteString(" ")
		}

		// builder.WriteString("%{A}")

		builder.WriteString("%{F-}")
	}

	fmt.Fprintln(os.Stdout, builder.String())
	// logrus.Debug(builder.String())
}

func (p polybarFormat) Close() {
}
