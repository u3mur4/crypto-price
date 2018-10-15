package exchange

import (
	"encoding/json"
	"net/http"
	"time"
)

var httpClient = http.Client{Timeout: 10 * time.Second}

func httpGetJSON(url string, v interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return err
	}
	return nil
}

func runEvery(d time.Duration, f func()) (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		run := true
		for run {
			nextTime := time.Now().Truncate(d).Add(d).Sub(time.Now())
			select {
			case <-time.After(nextTime):
				f()
			case <-done:
				run = false
			}
		}
	}()
	return done
}
