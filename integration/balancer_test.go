package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

type RespBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	numRequests := 3

	servers := make([]string, numRequests)

	addresses := generateAPIAddresses(numRequests)

	for i, addr := range addresses {
		servers[i] = getServerName(t, addr)
	}

	if servers[0] != servers[2] {
		t.Errorf("Different servers for the same address: got %s and %s", servers[0], servers[2])
	}

	checkResponseBody(t, "lospollosbrovaros")
}

func generateAPIAddresses(num int) []string {
	addresses := make([]string, num)
	for i := 0; i < num; i++ {
		addresses[i] = fmt.Sprintf("%s/api/v1/some-data", baseAddress)
	}
	return addresses
}

func getServerName(t *testing.T, addr string) string {
	resp, err := client.Get(addr)
	if err != nil {
		t.Error(err)
		return ""
	}

	defer resp.Body.Close()
	server := resp.Header.Get("lb-from")
	if server == "" {
		t.Errorf("Missing 'lb-from' header in response for request to address %s", addr)
	}
	return server
}

func checkResponseBody(t *testing.T, key string) {
	addr := fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, key)
	resp, err := client.Get(addr)
	if err != nil {
		t.Error(err)
		return
	}

	var body RespBody
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Error(err)
		return
	}

	if body.Key != key {
		t.Errorf("Expected %s, got %s", key, body.Key)
	}

	if body.Value == "" {
		t.Errorf("Expected a non-empty body.Value")
	}

	fmt.Println(body.Value)
}

func TestBalancer_NotFound(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	checkResponseStatusCode(t, "wrongKEY", http.StatusNotFound)
}

func checkResponseStatusCode(t *testing.T, key string, expectedStatusCode int) {
	addr := fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, key)
	resp, err := client.Get(addr)
	if err != nil {
		t.Error(err)
		return
	}

	if resp.StatusCode != expectedStatusCode {
		t.Errorf("Expected status code %d, got %d", expectedStatusCode, resp.StatusCode)
	}

	resp.Body.Close()
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	addr := fmt.Sprintf("%s/api/v1/some-data", baseAddress)
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(addr)
		if err != nil {
			b.Error(err)
			continue
		}

		resp.Body.Close()
	}
}
