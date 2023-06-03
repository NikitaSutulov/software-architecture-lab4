package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/NikitaSutulov/software-architecture-lab4/datastore"
	"github.com/NikitaSutulov/software-architecture-lab4/httptools"
	"github.com/NikitaSutulov/software-architecture-lab4/signal"
)

var port = flag.Int("port", 8083, "server port")

type RespBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ReqBody struct {
	Value string `json:"value"`
}

func handleDbRequests(Db *datastore.Db, rw http.ResponseWriter, req *http.Request) {
	url := req.URL.String()
	key := url[4:]

	switch req.Method {
	case "GET":
		handleGetRequest(Db, rw, key)
	case "POST":
		handlePostRequest(Db, rw, req, key)
	default:
		rw.WriteHeader(http.StatusBadRequest)
	}
}

func handleGetRequest(Db *datastore.Db, rw http.ResponseWriter, key string) {
	value, err := Db.Get(key)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(rw).Encode(RespBody{Key: key, Value: value}); err != nil {
		log.Println("Error encoding response: ", err)
	}
}

func handlePostRequest(Db *datastore.Db, rw http.ResponseWriter, req *http.Request, key string) {
	var body ReqBody

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := Db.Put(key, body.Value); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusCreated)
}

func main() {
	flag.Parse()
	h := http.NewServeMux()

	dir, err := ioutil.TempDir("", "temp-dir")
	if err != nil {
		log.Fatal(err)
	}

	Db, err := datastore.NewDb(dir, 45)
	if err != nil {
		log.Fatal(err)
	}
	defer Db.Close()

	h.HandleFunc("/db/", func(rw http.ResponseWriter, req *http.Request) {
		handleDbRequests(Db, rw, req)
	})

	server := httptools.CreateServer(*port, h)
	go server.Start()

	signal.WaitForTerminationSignal()
}
