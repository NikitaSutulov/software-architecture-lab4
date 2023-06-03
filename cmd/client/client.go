package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var target = flag.String("target", "http://localhost:8090", "request target")

func main() {
	flag.Parse()
	client := new(http.Client)
	client.Timeout = 10 * time.Second

	for range time.Tick(1 * time.Second) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=lospollosbrovaros", *target))
		if err == nil {
			log.Printf("response %d", resp.StatusCode)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			resp.Body.Close()
			fmt.Println(string(body))
		} else {
			log.Printf("error %s", err)
		}
	}
}
