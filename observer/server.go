package observer

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/u3mur4/crypto-price/exchange"
	"github.com/u3mur4/crypto-price/internal/logger"
)

type MarketAPIServer struct {
	markets    map[string]exchange.MarketDisplayInfo
	jsonOutput *JSONOutput
	log        *logrus.Entry
}

func NewMarketAPIServer() *MarketAPIServer {
	server := &MarketAPIServer{
		markets:    make(map[string]exchange.MarketDisplayInfo),
		jsonOutput: NewJSONOutput(),
		log:        logger.Log().WithField("observer", "market_api_server"),
	}

	rtr := mux.NewRouter()
	rtr.HandleFunc("/{key:.*}", server.handler).Methods("GET")

	http.Handle("/api", rtr)
	go http.ListenAndServe(":23232", nil)

	return server
}

func (j *MarketAPIServer) handler(w http.ResponseWriter, r *http.Request) {
	key := strings.ToLower(r.URL.Path[1:])
	if chart, ok := j.markets[key]; ok {
		w.Header().Set("Content-Type", "application/json")
		j.jsonOutput.Output = w
		j.jsonOutput.Update(chart)
		j.log.WithField("key", key).Debug("GET request for market info")
	} else {
		j.log.WithField("key", key).Warn("Market not found")
		w.WriteHeader(404)
	}
}

func (j *MarketAPIServer) Update(info exchange.MarketDisplayInfo) {
	j.markets[info.Market.Key()] = info
}
