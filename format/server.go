package format

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/u3mur4/crypto-price/exchange"
)

// NewServer displays the market as json format
func NewServer() Formatter {
	return &serverFormatter{
		charts:        make(map[string]exchange.Chart),
		jsonFormatter: NewJSON().(*jsonFormat),
	}
}

type serverFormatter struct {
	charts        map[string]exchange.Chart
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
	if chart, ok := j.charts[key]; ok {
		w.Header().Set("Content-Type", "application/json")
		j.jsonFormatter.Output = w
		j.jsonFormatter.Show(chart)
		// fmt.Printf("Response:\n%s", string(b))
	} else {
		w.WriteHeader(404)
	}
}

func (j *serverFormatter) Show(chart exchange.Chart) {
	key := chart.Exchange + ":" + chart.Base + "-" + chart.Quote
	key = strings.ToLower(key)
	j.charts[key] = chart
}

func (j *serverFormatter) Close() {
	j.jsonFormatter.Close()
}
