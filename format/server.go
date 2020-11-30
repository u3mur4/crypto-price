package format

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// NewServer displays the market as json format
func NewServer() Formatter {
	return &serverFormatter{
		markets: make(map[string]Market),
	}
}

type serverFormatter struct {
	markets map[string]Market
}

func (j *serverFormatter) Open() {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/{key:.*}", j.handler).Methods("GET")

	http.Handle("/", rtr)
	go http.ListenAndServe(":23232", nil)

}
func (j *serverFormatter) handler(w http.ResponseWriter, r *http.Request) {
	key := strings.ToLower(r.URL.Path[1:])
	// fmt.Println("New request for " + key)
	if m, ok := j.markets[key]; ok {
		b, _ := json.Marshal(&jsonMarket{
			Exchange: m.Exchange(),
			Base:     m.Base(),
			Quote:    m.Quote(),
			Open:     m.Open(),
			Price:    m.Price(),
			Percent:  percent(m),
			Color:    color(m).Hex(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		// fmt.Printf("Response:\n%s", string(b))
	} else {
		w.WriteHeader(404)
	}
}

func (j *serverFormatter) Show(m Market) {
	key := m.Exchange() + ":" + m.Base() + "-" + m.Quote()
	key = strings.ToLower(key)
	// fmt.Printf("New market update: %s\n", key)
	j.markets[key] = m
}

func (j *serverFormatter) Close() {

}
