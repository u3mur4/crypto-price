package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/u3mur4/crypto-price/exchange"
	"github.com/u3mur4/crypto-price/format"
)

var flags = struct {
	Format      string
	Template    string
	Satoshi     bool
	I3Bar       bool
	I3BarSort   string
	I3BarIcon   bool
	Polybar     bool
	PolybarSort string
	JSON        bool
	JSONLine    bool
	Server      bool
	Debug       bool
	Alert       bool
	// Throotle    float64
}{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "crypto-price [flags] {exchange:ticker}...",
	Short: "Realtime Crypto Price Tracker",
	Long:  `Realtime Crypto Price Tracker`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if flags.Debug {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.ErrorLevel)
		}

		f, _ := os.Create("/tmp/crypto-tracker.log")
		logrus.SetOutput(f)
		logrus.SetLevel(logrus.DebugLevel)

		aggregator := exchange.NewAggregator(exchange.Options{
			ConvertToSatoshi: flags.Satoshi,
		}, format.NewPolybar(format.PolybarConfig{}))

		formats := []format.Formatter{}

		if flags.JSON {
			formats = append(formats, format.NewJSON())
		} else if flags.Server {
			formats = append(formats, format.NewServer())
		} else if flags.Template != "" {
			formats = append(formats, format.NewTemplate(flags.Template))
		} else if flags.Polybar {
			formats = append(formats, format.NewPolybar(format.PolybarConfig{
				Sort: flags.PolybarSort,
				Icon: false,
			}))
		}

		if flags.Alert {
			formats = append(formats, format.NewAlert())
		}

		aggregator.SetFormatter(format.NewMulti(formats...))

		err := aggregator.Register(args...)
		if err != nil {
			logrus.WithError(err).Fatal("register error")
		}

		aggregator.Start()
	},
}

func init() {
	rootCmd.Flags().BoolVar(&flags.Debug, "debug", false, "Enable debug log")
	//
	rootCmd.Flags().BoolVar(&flags.Satoshi, "satoshi", false, "convert btc market price to satoshi")

	// format flags
	rootCmd.Flags().StringVarP(&flags.Template, "template", "t", "", "golang template format")

	rootCmd.Flags().BoolVar(&flags.Server, "server", false, "start a http server")

	rootCmd.Flags().BoolVar(&flags.I3Bar, "i3bar", false, "i3bar format")
	rootCmd.Flags().BoolVar(&flags.I3BarIcon, "i3bar-icon", false, "Enable icons. (https://github.com/AllienWorks/cryptocoins)")
	rootCmd.Flags().StringVar(&flags.I3BarSort, "i3bar-sort", "keep", "sort markets by change. values: keep, inc, dec")

	rootCmd.Flags().BoolVar(&flags.Polybar, "polybar", true, "polybar format")
	rootCmd.Flags().StringVar(&flags.PolybarSort, "polybar-sort", "keep", "sort markets by change. values: keep, inc, dec")

	rootCmd.Flags().BoolVar(&flags.JSON, "json", false, "json format")
	rootCmd.Flags().BoolVar(&flags.JSONLine, "jsonl", false, "json line format")
	rootCmd.Flags().BoolVar(&flags.Alert, "alert", false, "enable alert")
}

func main() {
	// rootCmd.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
