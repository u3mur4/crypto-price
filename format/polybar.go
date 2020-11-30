package format

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type PolybarConfig struct {
	Sort string
	Icon bool
}

type polybarFormat struct {
	markets map[string]Market
	config  PolybarConfig
	keys    []string
}

func NewPolybar(config PolybarConfig) Formatter {
	return &polybarFormat{
		markets: make(map[string]Market),
		config:  config,
	}
}

func (p polybarFormat) Open() {

}

func (p *polybarFormat) formatQuote(market Market) string {
	if strings.EqualFold(market.Quote(), "btc") {
		return "Ƀ"
	} else if strings.EqualFold(market.Quote(), "usd") {
		return "$"
	} else if strings.EqualFold(market.Quote(), "eur") {
		return "€"
	}
	return ""
}

func (i *polybarFormat) formatPrice(market Market) string {
	if strings.EqualFold(market.Quote(), "btc") {
		if market.Price() < 1 {
			return fmt.Sprintf("%.8f", market.Price())
		}
		return fmt.Sprintf("%.0f", market.Price())
	}
	return fmt.Sprintf("%.2f", market.Price())
}

func (i *polybarFormat) getIcon(market Market) string {
	name := strings.ToUpper(market.Base())

	if icon, ok := i3BarIcons[name+"-alt"]; ok {
		return icon
	}

	if icon, ok := i3BarIcons[name]; ok {
		return icon
	}
	return ""
}

func (i *polybarFormat) openTradingViewCmd(m Market) string {
	b := strings.Builder{}

	b.WriteString("chromium --newtab ")
	b.WriteString("https://www.tradingview.com/chart/?symbol=")
	b.WriteString(m.Exchange())
	b.WriteString(":")
	b.WriteString(m.Base())
	b.WriteString(m.Quote())
	b.WriteString(" --profile-directory=\"Profile 2\"")

	return "%{A1:" + strings.Replace(b.String(), ":", "\\:", -1) + ":}"
}

func (i *polybarFormat) Show(m Market) {
	key := m.Exchange() + m.Base() + m.Quote()

	// keep output consistent
	if _, ok := i.markets[key]; !ok {
		i.keys = append(i.keys, key)
	} else {
		// do not update if the price is the same
		if i.markets[key].Price() == m.Price() {
			return
		}
	}

	i.markets[key] = m

	// sort key by change if required
	if !strings.HasPrefix(i.config.Sort, "keep") && i.config.Sort != "" {
		s := byPercent{
			Keys:    i.keys,
			Markets: i.markets,
		}
		if strings.HasPrefix(i.config.Sort, "inc") {
			s.Increase = true
		}
		sort.Sort(&s)
	}

	// format all market
	builder := strings.Builder{}
	for _, k := range i.keys {
		market := i.markets[k]

		price := i.formatPrice(market)
		quote := i.formatQuote(market)
		// icon := i.getIcon(market)

		// // use icon or the base
		// 	tmp := fmt.Sprintf("<span foreground='%s'>%s: </span>", color(market).Hex(), strings.ToUpper(market.Base()))
		// 	builder.WriteString(tmp)
		// }

		// // print price
		// tmp := fmt.Sprintf("<span foreground='%s'>%s%s (%+.1f%%)</span> ", color(market).Hex(), price, quote, percent(market))
		// builder.WriteString(tmp)

		builder.WriteString("%{F")
		builder.WriteString(color(market).Hex())
		builder.WriteString("}")

		builder.WriteString(i.openTradingViewCmd(market))

		builder.WriteString(strings.ToUpper(market.Base()))
		builder.WriteString(": ")
		builder.WriteString(price)
		builder.WriteString(quote)
		builder.WriteString(fmt.Sprintf(" (%+.1f%%) ", percent(market)))

		builder.WriteString("%{A}")

		builder.WriteString("%{F-}")
	}

	fmt.Fprintln(os.Stdout, builder.String())
	// logrus.Debug(builder.String())
}

func (p polybarFormat) Close() {
}
