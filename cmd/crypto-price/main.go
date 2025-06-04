package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/u3mur4/crypto-price/exchange"
	"github.com/u3mur4/crypto-price/internal/logger"
	"github.com/u3mur4/crypto-price/observer"
)

var flags = struct {
	Template                  string
	Satoshi                   bool
	Polybar                   bool
	PolybarShortOnlyOnWeekend bool
	Waybar                    bool
	WaybarShortOnlyOnWeekend  bool
	JSON                      bool
	Server                    bool
	Debug                     bool
	Alert                     bool
}{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "crypto-price [flags] {exchange:ticker}...",
	Short: "Realtime Crypto Price Tracker",
	Long:  `Realtime Crypto Price Tracker`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if flags.Debug {
			logger.Setup("debug", false)
		} else {
			logger.Setup("error", false)
		}

		aggregator := exchange.NewAggregator(exchange.Options{
			ConvertToSatoshi: flags.Satoshi,
		}, observer.NewPolybarOutput(observer.PolybarConfig{}))

		observers := []exchange.Observer{}

		if flags.JSON {
			observers = append(observers, observer.NewJSONOutput())
		}
		if flags.Server {
			observers = append(observers, observer.NewMarketAPIServer())
		}
		if flags.Template != "" {
			observers = append(observers, observer.NewTemplateOutput(flags.Template))
		}
		if flags.Polybar {
			observers = append(observers, observer.NewPolybarOutput(observer.PolybarConfig{
				ShortOnlyOnWeekend: flags.PolybarShortOnlyOnWeekend,
			}))
		}
		if flags.Waybar {
			observers = append(observers, observer.NewWaybarOutput(observer.WaybarConfig{
				ShortOnlyOnWeekend: flags.WaybarShortOnlyOnWeekend,
			}))
		}

		if flags.Alert {
			alerter, err := observer.NewMarketAlerter()
			if err == nil {
				observers = append(observers, alerter)
			}
		}

		aggregator.AddObservers(observers...)

		err := aggregator.Register(args...)
		if err != nil {
			logrus.WithError(err).Fatal("register error")
		}

		aggregator.Start()
	},
}

func init() {
	rootCmd.Flags().BoolVar(&flags.Debug, "debug", true, "Enable debug log")
	rootCmd.Flags().BoolVar(&flags.Satoshi, "satoshi", false, "convert btc market price to satoshi")

	rootCmd.Flags().StringVarP(&flags.Template, "template", "t", "", "golang template format")

	rootCmd.Flags().BoolVar(&flags.Server, "server", false, "start a http server")

	rootCmd.Flags().BoolVar(&flags.Polybar, "polybar", false, "polybar format")
	rootCmd.Flags().BoolVar(&flags.PolybarShortOnlyOnWeekend, "polybar-weekend-short", false, "short display on weekend")

	rootCmd.Flags().BoolVar(&flags.Waybar, "waybar", false, "waybar format")
	rootCmd.Flags().BoolVar(&flags.WaybarShortOnlyOnWeekend, "waybar-weekend-short", false, "short display on weekend")

	rootCmd.Flags().BoolVar(&flags.JSON, "json", false, "json format")
	rootCmd.Flags().BoolVar(&flags.Alert, "alert", false, "enable alert")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
