package observer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-shellwords"
	"github.com/sirupsen/logrus"
	"github.com/u3mur4/crypto-price/exchange"
	"github.com/u3mur4/crypto-price/internal/logger"
)

type alertDefinition struct {
	ID          string    `json:"id"`
	Enabled     bool      `json:"enabled"`
	GracePeriod string    `json:"grace_period"` // Duration string, e.g. "5m" for 5 minutes which is the time to wait before triggering the alert again
	LastAlert   time.Time `json:"-"`
	Condition   string    `json:"condition"`
	Value       []float64 `json:"value"`
	Cmd         string    `json:"cmd"`
}

type MarketAlerter struct {
	alerts []*alertDefinition
	log    *logrus.Entry
}

func NewMarketAlerter() (*MarketAlerter, error) {
	alerter := &MarketAlerter{
		alerts: make([]*alertDefinition, 0),
		log:    logger.Log().WithField("observer", "alerter"),
	}

	err := alerter.load()
	if err != nil {
		return nil, err
	}

	go alerter.watchConfigFile()
	return alerter, nil
}

func (j *MarketAlerter) watchConfigFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		j.log.WithError(err).Error("failed to create file watcher, not watching for changes")
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
					// File modified, reload alerts
					j.load()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				j.log.WithError(err).Error("error watching alerts config file")
			}
		}
	}()

	err = watcher.Add(j.configPath())
	if err != nil {
		j.log.WithError(err).Error("failed to add file watcher for alerts config file")
	}
	<-done
}

func (j *MarketAlerter) configPath() string {
	userConfigDir, _ := os.UserConfigDir()
	alterConfigDir := path.Join(userConfigDir, "crypto-alerts")
	os.MkdirAll(alterConfigDir, 0755)
	return path.Join(alterConfigDir, "alerts.json")
}

func (j *MarketAlerter) load() error {
	byteValue, err := os.ReadFile(j.configPath())
	if err != nil {
		j.log.WithError(err).Error("failed to read alerts config file")
		return err
	}
	err = json.Unmarshal(byteValue, &j.alerts)
	if err != nil {
		j.log.WithError(err).Error("failed to unmarshal alerts config file")
		return err
	}

	j.log.WithField("count", len(j.alerts)).Info("loaded alerts")
	return nil
}

func (j *MarketAlerter) triggerAlertCmd(alert *alertDefinition) {
	args, err := shellwords.Parse(alert.Cmd)
	if err != nil {
		j.log.WithError(err).Errorf("cannot parse alert cmd(%s)", alert.Cmd)
		return
	}
	j.log.WithField("cmd", alert.Cmd).Info("triggering alert command")

	cmd := exec.Command(args[0], args[1:]...)
	err = cmd.Start()
	if err != nil {
		j.log.WithError(err).Error("cannot run alert cmd")
		return
	}

	alert.LastAlert = time.Now()
	cmd.Process.Release()
}

func (j *MarketAlerter) Update(info exchange.MarketDisplayInfo) {
	market := info.Market
	id := market.Exchange + ":" + market.Base + "-" + market.Quote

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
				if market.Candle.Percent() > alert.Value[0] {
					j.triggerAlertCmd(alert)
				}
			case "lt_percent":
				if market.Candle.Percent() < alert.Value[0] {
					j.triggerAlertCmd(alert)
				}
			case "gt_price":
				if market.Candle.Close > alert.Value[0] {
					j.triggerAlertCmd(alert)
				}
			case "lt_price":
				if market.Candle.Close < alert.Value[0] {
					j.triggerAlertCmd(alert)
				}
			}
		}
	}
}
