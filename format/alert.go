package format

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-shellwords"
	"github.com/u3mur4/crypto-price/exchange"
)

type alert struct {
	ID          string    `json:"id"`
	Enabled     bool      `json:"enabled"`
	GracePeriod string    `json:"grace_period"`
	LastAlert   time.Time `json:"-"`
	Condition   string    `json:"condition"`
	Value       []float64 `json:"value"`
	Cmd         string    `json:"cmd"`
}

type alertFormat struct {
	alerts []*alert
}

func NewAlert() Formatter {
	return &alertFormat{}
}

func (j *alertFormat) watchAlert() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					alerts, err := j.parseAlerts()
					if err == nil {
						j.alerts = alerts
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(j.alertPath())
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func (j *alertFormat) alertPath() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".crypto-alerts.json")
}

func (j *alertFormat) parseAlerts() (alerts []*alert, err error) {
	jsonFile, err := os.Open(j.alertPath())
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	print(string(byteValue))
	err = json.Unmarshal(byteValue, &alerts)
	if err != nil {
		return nil, err
	}

	fmt.Printf("found %d alerts\n", len(alerts))
	return alerts, nil
}

func (j *alertFormat) saveAlerts(alerts []*alert) {
	byteValue, _ := json.MarshalIndent(&j.alerts, "", "    ")
	ioutil.WriteFile(j.alertPath(), byteValue, 0644)
	return
}

func (j *alertFormat) Open() {
	alerts, err := j.parseAlerts()
	if err != nil {
		log.Println("error in open whil parsing alter config file:", err)
	}
	j.alerts = alerts
	go j.watchAlert()
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

	alert.LastAlert = time.Now()
	cmd.Process.Release()
}

func (j *alertFormat) Show(chart exchange.Chart) {
	id := chart.Exchange + ":" + chart.Base + "-" + chart.Quote
	for _, alert := range j.alerts {
		if strings.EqualFold(id, alert.ID) && alert.Enabled {
			gracePeriod, err := time.ParseDuration(alert.GracePeriod)
			if err == nil && !alert.LastAlert.IsZero() {
				fmt.Println(gracePeriod, alert.LastAlert)
				if time.Now().Before(alert.LastAlert.Add(gracePeriod)) {
					continue
				}
			}

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
