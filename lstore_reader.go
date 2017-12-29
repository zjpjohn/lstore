package lstore

import (
	"github.com/esdb/gocodec"
)

// Reader is not thread safe, can only be used from one goroutine
type Reader struct {
	store          *Store
	currentVersion *StoreVersion
	tailSeq        RowSeq    // seq to start next cache fill
	tailRows       rowsChunk // rows cache
	gocIter        *gocodec.Iterator
}

func (store *Store) NewReader() (*Reader, error) {
	reader := &Reader{
		store:    store,
		tailRows: rowsChunk{},
		gocIter:  gocodec.NewIterator(nil),
	}
	if _, err := reader.Refresh(); err != nil {
		return nil, err
	}
	return reader, nil
}

// Refresh has minimum cost of two cas read, one for store.latestVersion, one for tailSegment.tail
func (reader *Reader) Refresh() (bool, error) {
	latestVersion := reader.store.latest()
	defer latestVersion.Close()
	if reader.currentVersion == nil || latestVersion.tailSegment != reader.currentVersion.tailSegment {
		reader.tailSeq = latestVersion.tailSegment.StartSeq
		reader.tailRows = make(rowsChunk, 0, 4)
	}
	if reader.currentVersion != latestVersion {
		// when reader moves forward, older version has a chance to die
		if reader.currentVersion != nil {
			if err := reader.currentVersion.Close(); err != nil {
				return false, err
			}
		}
		latestVersion.Acquire()
		reader.currentVersion = latestVersion
	}
	return reader.currentVersion.tailSegment.read(reader)
}

func (reader *Reader) Close() error {
	return reader.currentVersion.Close()
}
