package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cryptoprice "github.com/u3mur4/crypto-price"
	"github.com/u3mur4/crypto-price/format"
)

var flags = struct {
	Format    string
	Template  string
	Satoshi   bool
	I3Bar     bool
	I3BarSort string
	I3BarIcon bool
	JSON      bool
	JSONLine  bool
	Debug     bool
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
		c := cryptoprice.NewClient(cryptoprice.Options{
			ConvertToSatoshi: flags.Satoshi,
		})

		if flags.JSON {
			c.SetFormatter(format.NewJSON())
		} else if flags.JSONLine {
			c.SetFormatter(format.NewJSONLine())
		} else if flags.Template != "" {
			c.SetFormatter(format.NewTemplate(flags.Template))
		} else if flags.I3Bar {
			c.SetFormatter(format.NewI3Bar(format.I3BarConfig{
				Sort: flags.I3BarSort,
				Icon: flags.I3BarIcon,
			}))
		}

		err := c.Register(args...)
		if err != nil {
			logrus.WithError(err).Fatal("register error")
		}

		c.Start()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVar(&flags.Debug, "debug", false, "Enable debug log")
	//
	rootCmd.Flags().BoolVar(&flags.Satoshi, "satoshi", false, "convert btc market price to satoshi")

	// format flags
	rootCmd.Flags().StringVarP(&flags.Template, "template", "t", "", "golang template format")
	rootCmd.Flags().BoolVar(&flags.I3Bar, "i3bar", true, "i3bar format")
	rootCmd.Flags().BoolVar(&flags.I3BarIcon, "i3bar-icon", false, "Enable icons. (https://github.com/AllienWorks/cryptocoins)")
	rootCmd.Flags().StringVar(&flags.I3BarSort, "i3bar-sort", "keep", "sort markets by change. values: keep, inc, dec")
	rootCmd.Flags().BoolVar(&flags.JSON, "json", false, "json format")
	rootCmd.Flags().BoolVar(&flags.JSONLine, "jsonl", false, "json line format")
}
