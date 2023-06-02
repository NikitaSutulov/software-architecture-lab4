package datastore

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEntry_Encode(t *testing.T) {
	encoder := Entry{"tK", "tV"}
	data := encoder.Encode()
	encoder.Decode(data)
	if encoder.GetLength() != 16 {
		t.Error("Incorrect length")
	}
	if encoder.key != "tK" {
		t.Error("Incorrect key")
	}
	if encoder.value != "tV" {
		t.Error("Incorrect value")
	}
}

func TestReadValue(t *testing.T) {
	encoder := Entry{"tK", "tV"}
	data := encoder.Encode()
	readData := bytes.NewReader(data)
	bReadData := bufio.NewReader(readData)
	value, err := readValue(bReadData)
	if err != nil {
		t.Fatal(err)
	}
	if value != "tV" {
		t.Errorf("Wrong value: [%s]", value)
	}
}
