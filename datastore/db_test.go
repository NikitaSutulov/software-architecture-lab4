package datastore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDb_Put(t *testing.T) {
	saveDirectory, err := ioutil.TempDir("", "testDir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(saveDirectory)

	dataBase, err := NewDb(saveDirectory, 45)
	if err != nil {
		t.Fatal(err)
	}
	defer dataBase.Close()

	pairs := [][]string{
		{"1", "v1"},
		{"2", "v2"},
		{"3", "v3"},
	}
	finalPath := filepath.Join(saveDirectory, outFileName+"0")
	outFile, err := os.Open(finalPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("check put and get methods", func(t *testing.T) {
		for _, pair := range pairs {
			err := dataBase.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Unable to place %s: %s.", pair[0], err)
			}
			actual, err := dataBase.Get(pair[0])
			if err != nil {
				t.Errorf("Unable to retrieve %s: %s", pair[0], err)
			}
			if actual != pair[1] {
				t.Errorf("Invalid value returned. Expected: %s, Actual: %s.", pair[1], actual)
			}
		}
	})

	outInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	expectedStateSize := outInfo.Size()

	t.Run("check increase file size test", func(t *testing.T) {
		for _, pair := range pairs {
			err := dataBase.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Unable to place %s: %s.", pair[0], err)
			}
		}
		t.Log(dataBase)
		outInfo, err := outFile.Stat()
		actualStateSize := outInfo.Size()
		if err != nil {
			t.Fatal(err)
		}
		if expectedStateSize != actualStateSize {
			t.Errorf("Size mismatch: Expected: %d, Actual: %d.", expectedStateSize, actualStateSize)
		}
	})

	t.Run("check creation new process test", func(t *testing.T) {
		if err := dataBase.Close(); err != nil {
			t.Fatal(err)
		}
		dataBase, err = NewDb(saveDirectory, 45)
		if err != nil {
			t.Fatal(err)
		}

		for _, pair := range pairs {
			actual, err := dataBase.Get(pair[0])
			if err != nil {
				t.Errorf("Unable to place %s: %s.", pair[1], err)
			}
			expected := pair[1]
			if actual != expected {
				t.Errorf("Invalid value returned. Expected: %s, Actual: %s.", expected, actual)
			}
		}
	})
}

func TestDb_Segmentation(t *testing.T) {
	saveDirectory, err := ioutil.TempDir("", "testDir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(saveDirectory)

	db, err := NewDb(saveDirectory, 35)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("check creation of new file", func(t *testing.T) {
		db.Put("1", "v1")
		db.Put("2", "v2")
		db.Put("3", "v3")
		db.Put("2", "v5")
		actualTwoFiles := len(db.segments)
		expected2Files := 2
		if actualTwoFiles != expected2Files {
			t.Errorf("An error occurred during segmentation. Expected 2 files, but received %d.", len(db.segments))
		}
	})

	t.Run("check starting segmentation", func(t *testing.T) {
		db.Put("4", "v4")
		actualTreeFiles := len(db.segments)
		expected3Files := 3
		if actualTreeFiles != expected3Files {
			t.Errorf("An error occurred during segmentation. Expected 3 files, but received %d.", len(db.segments))
		}

		time.Sleep(2 * time.Second)

		actualTwoFiles := len(db.segments)
		expected2Files := 2
		if actualTwoFiles != expected2Files {
			t.Errorf("An error occurred during segmentation. Expected 2 files, but received %d.", len(db.segments))
		}
	})

	t.Run("check not storing new values of duplicate keys", func(t *testing.T) {
		actual, _ := db.Get("2")
		expected := "v5"
		if actual != expected {
			t.Errorf("An error occurred during segmentation. Expected value: %s, Actual one: %s", expected, actual)
		}
	})

	t.Run("check szie", func(t *testing.T) {
		file, err := os.Open(db.segments[0].filePath)
		defer file.Close()

		if err != nil {
			t.Error(err)
		}
		inf, _ := file.Stat()
		actual := inf.Size()
		expected := int64(45)
		if actual != expected {
			t.Errorf("An error occurred during segmentation. Expected size %d, Actual one: %d", expected, actual)
		}
	})
}

func TestDb_Delete(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, 150)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// put some initial data
	err = db.Put("key1", "value1")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Put("key2", "value2")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Put("key3", "value3")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("delete operation", func(t *testing.T) {
		// delete a key
		db.Delete("key2")

		// try to get deleted key
		_, err = db.Get("key2")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound for deleted key, got: %v", err)
		}

		// confirm other key still present
		val, err := db.Get("key1")
		if err != nil {
			t.Errorf("Failed to get existing key: %v", err)
		}
		if val != "value1" {
			t.Errorf("Bad value returned expected value1, got %s", val)
		}
	})

	t.Run("delete non-existing key", func(t *testing.T) {
		// delete a non-existing key
		db.Delete("key4")

		// confirm that the non-existing key is still non-existing
		_, err = db.Get("key4")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound for non-existing key, got: %v", err)
		}
	})
}
