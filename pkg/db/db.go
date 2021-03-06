package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/desertbit/closer/v3"
	"github.com/rs/zerolog/log"

	"github.com/boltdb/bolt"
)

const (
	bucketRR = "rr"
)

var (
	ErrNotFound = errors.New("not found")
)

type DB struct {
	closer.Closer

	b *bolt.DB
}

func Open(cl closer.Closer, dbPath string) (d *DB, err error) {
	b, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return
	}
	cl.OnClosing(b.Close)

	d = &DB{
		Closer: cl,
		b:      b,
	}

	err = d.init()
	if err != nil {
		return
	}

	go d.expireRoutine()
	return
}

func (d *DB) init() error {
	// Initialize if not already.
	return d.b.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketRR))
		return err
	})
}

func (d *DB) SetRecord(key string, record Record) (err error) {
	v, err := record.MarshalMsg(nil)
	if err != nil {
		return
	}

	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRR))
		if b == nil {
			return fmt.Errorf("bucket does not exist: %v", bucketRR)
		}
		return b.Put([]byte(key), v)
	})
}

func (d *DB) GetRecord(key string) (r Record, err error) {
	err = d.b.View(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(bucketRR))
		if b == nil {
			return fmt.Errorf("bucket does not exist: %v", bucketRR)
		}

		v := b.Get([]byte(key))
		if len(v) == 0 {
			return ErrNotFound
		}

		_, err = r.UnmarshalMsg(v)
		return
	})
	if err != nil {
		return
	}

	// Check if already expired.
	if r.Expires > 0 && r.Expires < time.Now().Unix() {
		err = ErrNotFound
	}
	return
}

func (d *DB) DeleteRecord(key string) (err error) {
	return d.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRR))
		if b == nil {
			return fmt.Errorf("bucket does not exist: %v", bucketRR)
		}
		return b.Delete([]byte(key))
	})
}

func (d *DB) deleteExpiredRecords() (err error) {
	var (
		r    Record
		list [][]byte

		ts = time.Now().Unix()
	)

	err = d.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRR))
		if b == nil {
			return fmt.Errorf("bucket does not exist: %v", bucketRR)
		}

		return b.ForEach(func(k, v []byte) (err error) {
			_, err = r.UnmarshalMsg(v)
			if err != nil {
				// Delete invalid entries.
				list = append(list, k)
				err = nil
				return
			}

			if r.Expires > 0 && r.Expires < ts {
				list = append(list, k)
			}
			return
		})
	})

	if len(list) == 0 {
		return
	}

	for _, k := range list {
		log.Info().Str("key", string(k)).Msg("dropping expired record")
	}

	return d.b.Update(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(bucketRR))
		if b == nil {
			return fmt.Errorf("bucket does not exist: %v", bucketRR)
		}

		for _, k := range list {
			err = b.Delete(k)
			if err != nil {
				return
			}
		}
		return
	})
}

func (d *DB) expireRoutine() {
	defer d.Close_()

	var (
		closingChan = d.ClosingChan()
	)

	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()

	for {
		select {
		case <-closingChan:
			return
		case <-t.C:
		}

		err := d.deleteExpiredRecords()
		if err != nil {
			log.Error().Err(err).Msg("failed to delete expired records")
			return // This is fatal.
		}
	}
}
