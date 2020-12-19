// +build postgresql

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/raphaelreyna/latte/internal/server"
)

type Database struct {
	db *gorm.DB
}

type Blob struct {
	ID    int    `gorm:"primary_key"`
	UID   string `gorm:"unique_index"`
	Bytes []byte
}

func init() {
	var err error
	db, err = newDB()
	if err != nil {
		log.Fatalf("fatal error occurred while creating database connection pool: %v", err)
	}
}

func newDB() (server.DB, error) {
	host := os.Getenv("LATTE_DB_HOST")
	port := os.Getenv("LATTE_DB_PORT")
	name := os.Getenv("LATTE_DB_NAME")
	username := os.Getenv("LATTE_DB_USERNAME")
	password := os.Getenv("LATTE_DB_PASSWORD")
	ssl := os.Getenv("LATTE_DB_SSL")

	connstr := "host=%s port=%s dbname=%s user=%s password=%s"
	connstr = connstr + " sslmode=%s connect_timeout=10"
	connstr = fmt.Sprintf(connstr,
		host, port, name,
		username, password, ssl,
	)

	var db Database
	var err error
	db.db, err = gorm.Open("postgres", connstr)
	if err != nil {
		return nil, err
	}
	db.db.AutoMigrate(&Blob{})
	return &db, nil
}

func (db *Database) Store(ctx context.Context, uid string, i interface{}) error {
	var err error
	blob := Blob{UID: uid}
	switch i.(type) {
	case []byte:
		blob.Bytes = i.([]byte)
	case io.ReadCloser:
		rc := i.(io.ReadCloser)
		blob.Bytes, err = ioutil.ReadAll(rc)
		if err != nil {
			return err
		}
		if err = rc.Close(); err != nil {
			return err
		}
	default:
		return errors.New("can only store []byte or io.ReadCloser contents")
	}
	return db.db.Create(&blob).Error
}

func (db *Database) Fetch(ctx context.Context, uid string) (interface{}, error) {
	var blob Blob
	res := db.db.First(&blob, "uid = ?", uid)
	if err := res.Error; res.RecordNotFound() {
		return nil, &server.NotFoundError{}
	} else if err != nil {
		return nil, err
	}
	return blob.Bytes, nil
}

func (db *Database) Ping(ctx context.Context) error {
	return db.db.DB().PingContext(ctx)
}

// AddFileAs allows *Database to satisfy the recon.Source interface (github.com/raphaelreyna/go-recon)
func (db *Database) AddFileAs(name, destination string, perm os.FileMode) error {
	var blob Blob
	res := db.db.First(&blob, "uid = ?", name)
	if err := res.Error; res.RecordNotFound() {
		return nil, &server.NotFoundError{}
	} else if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(blob.Bytes)
	return err
}
