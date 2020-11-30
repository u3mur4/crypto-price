package format

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type i3BarFormat struct {
	Output  io.Writer
	markets map[string]Market
	keys    []string
	printer *message.Printer
	config  I3BarConfig
}

type I3BarConfig struct {
	Sort string
	Icon bool
}

type byPercent struct {
	Keys     []string
	Markets  map[string]Market
	Increase bool
}

func (a byPercent) Len() int { return len(a.Keys) }
func (a byPercent) Less(i, j int) bool {
	f := percent(a.Markets[a.Keys[i]])
	s := percent(a.Markets[a.Keys[j]])
	if a.Increase {
		return f < s
	}
	return f > s
}
func (a *byPercent) Swap(i, j int) { a.Keys[i], a.Keys[j] = a.Keys[j], a.Keys[i] }

// NewI3Bar displays the market as i3bar format
func NewI3Bar(config I3BarConfig) Formatter {
	return &i3BarFormat{
		Output:  os.Stdout,
		markets: make(map[string]Market),
		printer: message.NewPrinter(language.English),
		config:  config,
	}
}

func (i i3BarFormat) Open() {
	// handle user input
	go func() {
		for {
			var input string
			_, err := fmt.Scanln(&input)
			if err != nil {
				break
			}
			logrus.WithField("input", input).Debug("new i3 input")
			// left click
			if input == "1" {
				fmt.Println("<span>UPDATE</span>")
				cmd := exec.Command("chromium", "--new-tab")
				cmd.Run()
			}
		}
	}()
}

func (i *i3BarFormat) formatQuote(market Market) string {
	if strings.EqualFold(market.Quote(), "btc") {
		return "Ƀ"
	} else if strings.EqualFold(market.Quote(), "usd") {
		return "$"
	} else if strings.EqualFold(market.Quote(), "eur") {
		return "€"
	}
	return ""
}

func (i *i3BarFormat) formatPrice(market Market) string {
	if strings.EqualFold(market.Quote(), "btc") {
		if market.Price() < 1 {
			return fmt.Sprintf("%.8f", market.Price())
		}
		return fmt.Sprintf("%.0f", market.Price())
	}
	return fmt.Sprintf("%.2f", market.Price())
}

func (i *i3BarFormat) getIcon(market Market) string {
	name := strings.ToUpper(market.Base())

	if icon, ok := i3BarIcons[name+"-alt"]; ok {
		return icon
	}

	if icon, ok := i3BarIcons[name]; ok {
		return icon
	}
	return ""
}

func (i *i3BarFormat) Show(m Market) {
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
		icon := i.getIcon(market)

		// use icon or the base
		if i.config.Icon && icon != "" {
			tmp := fmt.Sprintf("<span font='cryptocoins' foreground='%s'>%s</span><span>: </span>", color(market).Hex(), icon)
			builder.WriteString(tmp)
		} else {
			tmp := fmt.Sprintf("<span foreground='%s'>%s: </span>", color(market).Hex(), strings.ToUpper(market.Base()))
			builder.WriteString(tmp)
		}

		// print price
		tmp := fmt.Sprintf("<span foreground='%s'>%s%s (%+.1f%%)</span> ", color(market).Hex(), price, quote, percent(market))
		builder.WriteString(tmp)

		// f := fmt.Sprintf("<span foreground='%s'>%s: %s%s (%+.1f%%)</span> ", color(market).Hex(), strings.ToUpper(market.Base()), price, quote, percent(market))
		// builder.WriteString(f)

	}

	// if len(i.markets) > 0 {
	// 	builder.WriteString("\n")
	// }
	fmt.Fprintln(os.Stdout, builder.String())
	logrus.Debug(builder.String())
}

func (i i3BarFormat) Close() {
}
