package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/NikitaSutulov/software-architecture-lab4/httptools"
	"github.com/NikitaSutulov/software-architecture-lab4/signal"
)

var port = flag.Int("port", 8080, "server port")
var report = make(Report)

const (
	confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
	confHealthFailure    = "CONF_HEALTH_FAILURE"
	dbUrl                = "http://db:8083/db/"
)

type ReqBody struct {
	Value string `json:"value"`
}

type RespBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func main() {
	flag.Parse()
	h := http.NewServeMux()
	client := http.DefaultClient

	h.HandleFunc("/health", healthHandler)
	h.HandleFunc("/api/v1/some-data", someDataHandler(client))
	h.HandleFunc("/api/v1/some-data2", someData2Handler)
	h.HandleFunc("/api/v1/some-data3", someData3Handler)

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	go server.Start()

	buff := new(bytes.Buffer)
	body := ReqBody{Value: time.Now().Format(time.RFC3339)}
	json.NewEncoder(buff).Encode(body)

	res, err := client.Post(fmt.Sprintf("%slospollosbrovaros", dbUrl), "application/json", buff)
	if err != nil {
		log.Fatalf("Post request failed: %v", err)
	}

	defer res.Body.Close()

	signal.WaitForTerminationSignal()
}

func healthHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("content-type", "text/plain")
	if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("FAILURE"))
	} else {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("OK"))
	}
}

func someDataHandler(client *http.Client) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		resp, err := client.Get(fmt.Sprintf("%s/%s", dbUrl, key))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		statusOk := resp.StatusCode >= 200 && resp.StatusCode < 300

		if !statusOk {
			rw.WriteHeader(resp.StatusCode)
			return
		}

		applyDelay()
		report.Process(r)

		var body RespBody
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		jsonResponseHandler(rw, body)
	}
}

func someData2Handler(rw http.ResponseWriter, r *http.Request) {
	applyDelay()
	report.Process(r)

	jsonResponseHandler(rw, []string{"2", "3"})
}

func someData3Handler(rw http.ResponseWriter, r *http.Request) {
	applyDelay()
	report.Process(r)

	jsonResponseHandler(rw, []string{"3", "4"})
}

func jsonResponseHandler(rw http.ResponseWriter, body interface{}) {
	rw.Header().Set("content-type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(body)
}

func applyDelay() {
	respDelayString := os.Getenv(confResponseDelaySec)
	if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
		time.Sleep(time.Duration(delaySec) * time.Second)
	}
}
