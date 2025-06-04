package observer

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/u3mur4/crypto-price/exchange"
)

// NewServer displays the market as json format
func NewMarketAPIServer() *MarketAPIServer {
	return &MarketAPIServer{
		markets:       make(map[string]exchange.MarketDisplayInfo),
		jsonOutput: NewJSONOutput(),
	}
}

type MarketAPIServer struct {
	markets       map[string]exchange.MarketDisplayInfo
	jsonOutput *JSONOutput
}

func (j *MarketAPIServer) Open() {
	j.jsonOutput.Open()

	rtr := mux.NewRouter()
	rtr.HandleFunc("/{key:.*}", j.handler).Methods("GET")

	http.Handle("/", rtr)
	go http.ListenAndServe(":23232", nil)

}
func (j *MarketAPIServer) handler(w http.ResponseWriter, r *http.Request) {
	key := strings.ToLower(r.URL.Path[1:])
	// fmt.Println("New request for " + key)
	if chart, ok := j.markets[key]; ok {
		w.Header().Set("Content-Type", "application/json")
		j.jsonOutput.Output = w
		j.jsonOutput.Show(chart)
		// fmt.Printf("Response:\n%s", string(b))
	} else {
		w.WriteHeader(404)
	}
}

func (j *MarketAPIServer) Show(info exchange.MarketDisplayInfo) {
	market := info.Market
	key := market.Exchange + ":" + market.Base + "-" + market.Quote
	key = strings.ToLower(key)
	j.markets[key] = info
}

func (j *MarketAPIServer) Close() {
	j.jsonOutput.Close()
}
