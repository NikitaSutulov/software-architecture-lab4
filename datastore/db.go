package datastore

import (
	"bufio"
	_ "bufio"
	"encoding/binary"
	_ "encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	outFileName  = "current-data"
	bufSize      = 8192
	deleteMarker = "DELETE"
)

type hashIndex map[string]int64

type IndexOp struct {
	isWrite bool
	key     string
	index   int64
}

type DeleteOp struct {
	key  string
	resp chan error
}

type KeyPosition struct {
	segment  *Segment
	position int64
}

type Db struct {
	out              *os.File
	outPath          string
	outOffset        int64
	dir              string
	segmentSize      int64
	lastSegmentIndex int
	indexOps         chan IndexOp
	keyPositions     chan *KeyPosition
	putOps           chan Entry
	deleteOps        chan DeleteOp
	putDone          chan error
	index            hashIndex
	segments         []*Segment
}

type Segment struct {
	outOffset int64
	index     hashIndex
	filePath  string
}

var (
	ErrNotFound = fmt.Errorf("record does not exist")
)

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments:     make([]*Segment, 0),
		dir:          dir,
		segmentSize:  segmentSize,
		indexOps:     make(chan IndexOp),
		keyPositions: make(chan *KeyPosition),
		putOps:       make(chan Entry),
		putDone:      make(chan error),
		deleteOps:    make(chan DeleteOp),
	}

	if err := db.createSegment(); err != nil {
		return nil, err
	}

	if err := db.recover(); err != nil && err != io.EOF {
		return nil, err
	}

	db.startDeleteRoutine()
	db.startIndexRoutine()
	db.startPutRoutine()

	return db, nil
}

func (db *Db) Close() error {
	return db.out.Close()
}

func (db *Db) startIndexRoutine() {
	go func() {
		for op := range db.indexOps {
			if op.isWrite {
				db.setKey(op.key, op.index)
			} else {
				s, p, err := db.getSegmentAndPos(op.key)
				if err != nil {
					db.keyPositions <- nil
				} else {
					db.keyPositions <- &KeyPosition{s, p}
				}
			}
		}
	}()
}

func (db *Db) startPutRoutine() {
	go func() {
		for {
			e := <-db.putOps
			length := e.GetLength()
			stat, err := db.out.Stat()
			if err != nil {
				db.putDone <- err
				continue
			}
			if stat.Size()+length > db.segmentSize {
				err := db.createSegment()
				if err != nil {
					db.putDone <- err
					continue
				}
			}
			n, err := db.out.Write(e.Encode())
			if err == nil {
				db.indexOps <- IndexOp{
					isWrite: true,
					key:     e.key,
					index:   int64(n),
				}
			}
			db.putDone <- nil
		}
	}()
}

func (db *Db) startDeleteRoutine() {
	go func() {
		for op := range db.deleteOps {
			s, p, err := db.getSegmentAndPos(op.key)
			if err != nil {
				op.resp <- err
				continue
			}
			s.index[op.key] = p
			db.out.Write([]byte(deleteMarker))
			delete(s.index, op.key)
			op.resp <- nil
		}
	}()
}

func (db *Db) createSegment() error {
	filePath := db.generateNewFileName()
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	newSegment := &Segment{
		filePath: filePath,
		index:    make(hashIndex),
	}

	db.out = f
	db.outOffset = 0
	db.outPath = filePath
	db.segments = append(db.segments, newSegment)
	if len(db.segments) >= 3 {
		db.performOldSegmentsCompaction()
	}

	return nil
}

func (db *Db) generateNewFileName() string {
	result := filepath.Join(db.dir, fmt.Sprintf("%s%d", outFileName, db.lastSegmentIndex))
	db.lastSegmentIndex++
	return result
}

func (db *Db) performOldSegmentsCompaction() {
	go func() {
		filePath := db.generateNewFileName()
		newSegment := &Segment{
			filePath: filePath,
			index:    make(hashIndex),
		}
		var offset int64
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			return
		}
		lastSegmentIndex := len(db.segments) - 2
		for i := 0; i <= lastSegmentIndex; i++ {
			s := db.segments[i]
			for key, index := range s.index {
				if i < lastSegmentIndex {
					isInNewerSegments := findKeyInSegments(db.segments[i+1:lastSegmentIndex+1], key)
					if isInNewerSegments {
						continue
					}
				}
				value, _ := s.getFromSegment(index)
				if value == deleteMarker {
					continue
				}
				e := Entry{
					key:   key,
					value: value,
				}
				n, err := f.Write(e.Encode())
				if err == nil {
					newSegment.index[key] = offset
					offset += int64(n)
				}
			}
		}
		db.segments = []*Segment{newSegment, db.getLastSegment()}
	}()
}

func findKeyInSegments(segments []*Segment, key string) bool {
	for _, s := range segments {
		if _, ok := s.index[key]; ok {
			return true
		}
	}
	return false
}

func (db *Db) recover() error {
	var err error
	var buf [bufSize]byte

	in := bufio.NewReaderSize(db.out, bufSize)
	for err == nil {
		var (
			header, data []byte
			n            int
		)
		header, err = in.Peek(bufSize)
		if err == io.EOF {
			if len(header) == 0 {
				return err
			}
		} else if err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(header)

		if size < bufSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}
		n, err = in.Read(data)

		if err == nil {
			if n != int(size) {
				return fmt.Errorf("corrupted file")
			}

			var e Entry
			e.Decode(data)
			db.setKey(e.key, int64(n))
		}
	}
	return err
}

func (db *Db) setKey(key string, n int64) {
	db.getLastSegment().index[key] = db.outOffset
	db.outOffset += n
}

func (db *Db) getSegmentAndPos(key string) (*Segment, int64, error) {
	for i := range db.segments {
		s := db.segments[len(db.segments)-i-1]
		pos, ok := s.index[key]
		if ok {
			return s, pos, nil
		}
	}

	return nil, 0, ErrNotFound
}

func (db *Db) getPos(key string) *KeyPosition {
	op := IndexOp{
		isWrite: false,
		key:     key,
	}
	db.indexOps <- op
	return <-db.keyPositions
}

func (db *Db) Get(key string) (string, error) {
	keyPos := db.getPos(key)
	if keyPos == nil {
		return "", ErrNotFound
	}
	value, err := keyPos.segment.getFromSegment(keyPos.position)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (db *Db) Put(key, value string) error {
	e := Entry{
		key:   key,
		value: value,
	}
	db.putOps <- e
	return <-db.putDone
}

func (db *Db) Delete(key string) error {
	resp := make(chan error)
	db.deleteOps <- DeleteOp{key: key, resp: resp}
	err := <-resp
	close(resp)
	return err
}

func (db *Db) getLastSegment() *Segment {
	return db.segments[len(db.segments)-1]
}

func (s *Segment) getFromSegment(position int64) (string, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	value, err := readValue(reader)
	if err != nil {
		return "", err
	}
	return value, nil
}
