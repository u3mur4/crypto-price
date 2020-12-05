package format

import (
	"fmt"
	"os"
	"strings"

	"github.com/u3mur4/crypto-price/exchange"
)

type PolybarConfig struct {
	Sort string
	Icon bool
}

type polybarFormat struct {
	charts map[string]exchange.Chart
	config PolybarConfig
	keys   []string
}

func NewPolybar(config PolybarConfig) Formatter {
	return &polybarFormat{
		charts: make(map[string]exchange.Chart),
		config: config,
	}
}

func (p polybarFormat) Open() {
}

func (p *polybarFormat) formatQuote(chart exchange.Chart) string {
	if strings.EqualFold(chart.Quote, "btc") {
		// return "Ƀ"
		return ""
	} else if strings.EqualFold(chart.Quote, "usd") || strings.EqualFold(chart.Quote, "usdt") {
		return "$"
	} else if strings.EqualFold(chart.Quote, "eur") {
		return "€"
	}
	return ""
}

func (i *polybarFormat) formatPrice(chart exchange.Chart) string {
	if strings.EqualFold(chart.Quote, "btc") {
		if chart.Candle.Close < 1 {
			return fmt.Sprintf("%.8f", chart.Candle.Close)
		}
		return fmt.Sprintf("%.0f", chart.Candle.Close)
	}
	return fmt.Sprintf("%.0f", chart.Candle.Close)
}

func (i *polybarFormat) openTradingViewCmd(chart exchange.Chart) string {
	b := strings.Builder{}

	b.WriteString("chromium --newtab ")
	b.WriteString("https://www.tradingview.com/chart/?symbol=")
	b.WriteString(chart.Exchange)
	b.WriteString(":")
	b.WriteString(chart.Base)
	b.WriteString(chart.Quote)
	b.WriteString(" --profile-directory=\"Profile 2\"")

	return "%{A1:" + strings.Replace(b.String(), ":", "\\:", -1) + ":}"
}

func (i *polybarFormat) Show(chart exchange.Chart) {
	key := chart.Exchange + chart.Base + chart.Quote

	// keep output consistent
	if _, ok := i.charts[key]; !ok {
		i.keys = append(i.keys, key)
	}
	i.charts[key] = chart

	// format all market
	builder := strings.Builder{}
	for _, k := range i.keys {
		chart := i.charts[k]

		price := i.formatPrice(chart)
		quote := i.formatQuote(chart)
		// icon := i.getIcon(market)

		// // use icon or the base
		// 	tmp := fmt.Sprintf("<span foreground='%s'>%s: </span>", color(market).Hex(), strings.ToUpper(market.Base()))
		// 	builder.WriteString(tmp)
		// }

		// // print price
		// tmp := fmt.Sprintf("<span foreground='%s'>%s%s (%+.1f%%)</span> ", color(market).Hex(), price, quote, percent(market))
		// builder.WriteString(tmp)

		builder.WriteString("%{F")
		builder.WriteString(color(chart.Candle).Hex())
		builder.WriteString("}")

		builder.WriteString(i.openTradingViewCmd(chart))

		builder.WriteString(strings.ToUpper(chart.Base))
		builder.WriteString(": ")
		builder.WriteString(quote)
		builder.WriteString(price)
		builder.WriteString(fmt.Sprintf(" (%+.1f%%) ", chart.Candle.Percent()))

		builder.WriteString("%{A}")

		builder.WriteString("%{F-}")
	}

	fmt.Fprintln(os.Stdout, builder.String())
	// logrus.Debug(builder.String())
}

func (p polybarFormat) Close() {
}
