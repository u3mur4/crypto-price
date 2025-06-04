package observer

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/u3mur4/crypto-price/exchange"
)

// NewServer displays the market as json format
func NewServer() Formatter {
	return &serverFormatter{
		markets:        make(map[string]exchange.MarketDisplayInfo),
		jsonFormatter: NewJSON().(*jsonFormat),
	}
}

type serverFormatter struct {
	markets        map[string]exchange.MarketDisplayInfo
	jsonFormatter *jsonFormat
}

func (j *serverFormatter) Open() {
	j.jsonFormatter.Open()

	rtr := mux.NewRouter()
	rtr.HandleFunc("/{key:.*}", j.handler).Methods("GET")

	http.Handle("/", rtr)
	go http.ListenAndServe(":23232", nil)

}
func (j *serverFormatter) handler(w http.ResponseWriter, r *http.Request) {
	key := strings.ToLower(r.URL.Path[1:])
	// fmt.Println("New request for " + key)
	if chart, ok := j.markets[key]; ok {
		w.Header().Set("Content-Type", "application/json")
		j.jsonFormatter.Output = w
		j.jsonFormatter.Show(chart)
		// fmt.Printf("Response:\n%s", string(b))
	} else {
		w.WriteHeader(404)
	}
}

func (j *serverFormatter) Show(info exchange.MarketDisplayInfo) {
	market := info.Market
	key := market.Exchange + ":" + market.Base + "-" + market.Quote
	key = strings.ToLower(key)
	j.markets[key] = info
}

func (j *serverFormatter) Close() {
	j.jsonFormatter.Close()
}
