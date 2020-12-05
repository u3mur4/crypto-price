package format

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/u3mur4/crypto-price/exchange"
)

type alert struct {
	ID        string    `json:"id"`
	Enabled   bool      `json:"enabled"`
	Condition string    `json:"condition"`
	Value     []float64 `json:"value"`
	Cmd       string    `json:"cmd"`
}

type alertFormat struct {
	alerts []*alert
}

func NewAlert() Formatter {
	return &alertFormat{}
}

func (j *alertFormat) alertPath() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".crypto-alerts.json")
}

func (j *alertFormat) parseAlerts() (alerts []*alert) {
	jsonFile, err := os.Open(j.alertPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot parse alerts file")
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &alerts)
	fmt.Printf("found %d alerts", len(alerts))
	return
}

func (j *alertFormat) saveAlerts(alerts []*alert) {
	byteValue, _ := json.MarshalIndent(&j.alerts, "", "    ")
	ioutil.WriteFile(j.alertPath(), byteValue, 0644)
	return
}

func (j *alertFormat) Open() {
	j.alerts = j.parseAlerts()
}

func (j *alertFormat) triggerAlert(alert *alert) {
	args, err := shellwords.Parse(alert.Cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot parse alert cmd(%s): %v\n", alert.Cmd, err)
		return
	}
	fmt.Fprintf(os.Stderr, "running alert cmd: %s\n", alert.Cmd)

	cmd := exec.Command(args[0], args[1:]...)
	err = cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot run alert cmd(%s): %v\n", alert.Cmd, err)
		return
	}

	cmd.Process.Release()
	alert.Enabled = false
	j.saveAlerts(j.alerts)
}

func (j *alertFormat) Show(chart exchange.Chart) {
	id := chart.Exchange + ":" + chart.Base + "-" + chart.Quote
	for _, alert := range j.alerts {
		if strings.EqualFold(id, alert.ID) && alert.Enabled {
			switch alert.Condition {
			case "gt_percent":
				if chart.Candle.Percent() > alert.Value[0] {
					j.triggerAlert(alert)
				}
			case "lt_percent":
				if chart.Candle.Percent() < alert.Value[0] {
					j.triggerAlert(alert)
				}
			case "gt_price":
				if chart.Candle.Close > alert.Value[0] {
					j.triggerAlert(alert)
				}
			case "lt_price":
				if chart.Candle.Close < alert.Value[0] {
					j.triggerAlert(alert)
				}
			}
		}
	}
}

func (j *alertFormat) Close() {
	j.saveAlerts(j.alerts)
}
