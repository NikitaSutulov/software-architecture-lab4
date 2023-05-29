package integration

import (
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

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	numRequests := 3

	addresses := []string{
		fmt.Sprintf("%s/api/v1/some-data", baseAddress),
		fmt.Sprintf("%s/api/v1/some-data2", baseAddress),
		fmt.Sprintf("%s/api/v1/some-data", baseAddress),
	}

	servers := make([]string, numRequests)

	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(addresses[i])
		if err != nil {
			t.Error(err)
			continue
		}
		if resp != nil {
			defer resp.Body.Close()
			server := resp.Header.Get("lb-from")
			if server == "" {
				t.Errorf("Missing 'lb-from' header in response for request %d", i)
			}
			servers[i] = server
		} else {
			t.Errorf("Response is nil for request %d", i)
		}
	}

	if servers[0] != servers[2] {
		t.Errorf("Different servers for the same address: got %s and %s", servers[0], servers[2])
	}
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			b.Error(err)
		}
		defer resp.Body.Close()
	}
}
