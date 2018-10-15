package cmd

import (
	"fmt"
	"os"

	"github.com/u3mur4/crypto-price/cryptotracker/format"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/u3mur4/crypto-price/cryptotracker"
)

var flags = struct {
	Format   string
	Template string
	Satoshi  bool
	I3Bar    bool
	JSON     bool
	JSONLine bool
	// Throotle    float64
}{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "crypto-price [flags] {exchange:ticker}... ",
	Short: "Real Time Crypto Price Tracker",
	Long:  `Real Time Crypto Price Tracker`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := cryptotracker.NewClient(cryptotracker.Options{
			ConvertToSatoshi: flags.Satoshi,
		})

		if flags.JSON {
			c.Formatter = format.NewJSON()
		} else if flags.JSONLine {
			c.Formatter = format.NewJSONLine()
		} else if flags.Template != "" {
			c.Formatter = format.NewTemplate(flags.Template)
		} else if flags.I3Bar {
			c.Formatter = format.NewI3Bar()
		}

		err := c.Register(args...)
		if err != nil {
			logrus.WithError(err).Fatal("register error")
		}

		c.Run()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	//
	rootCmd.Flags().BoolVar(&flags.Satoshi, "satoshi", false, "Convert btc market price to satoshi")

	// format flags
	rootCmd.Flags().StringVarP(&flags.Template, "template", "t", "", "golang template format")
	rootCmd.Flags().BoolVar(&flags.I3Bar, "i3bar", true, "i3bar format")
	rootCmd.Flags().BoolVar(&flags.JSON, "json", false, "JSON format")
	rootCmd.Flags().BoolVar(&flags.JSONLine, "jsonl", false, "JSON line format")
}