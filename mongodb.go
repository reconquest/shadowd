package main

import (
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/seletskiy/hierr"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type mongodb struct {
	dsn      string
	hashTTL  time.Duration
	session  *mgo.Session
	database *mgo.Database
	shadows  *mgo.Collection
	keys     *mgo.Collection
	clients  *mgo.Collection
}

func (db *mongodb) GetPublicKeys(token string) (string, error) {
	var docs []map[string]interface{}
	err := db.keys.Find(bson.M{"token": token}).All(&docs)
	if err != nil {
		return "", hierr.Errorf(
			err, "can't obtain public keys from database",
		)
	}

	keys := []string{}
	for _, doc := range docs {
		keys = append(keys, doc["key"].(string))
	}

	return strings.Join(keys, "\n"), nil
}

func (db *mongodb) AddPublicKey(
	token string, key []byte, truncate bool,
) error {
	if truncate {
		_, err := db.keys.RemoveAll(bson.M{"token": token})
		if err != nil {
			return hierr.Errorf(
				err, "can't remove public keys",
			)
		}
	}

	err := db.keys.Insert(bson.M{"token": token, "key": string(key)})
	if err != nil {
		return hierr.Errorf(
			err, "can't add key to database",
		)
	}

	return nil
}

func (db *mongodb) AddHashTable(token string, table []string) error {
	_, err := db.shadows.RemoveAll(bson.M{"token": token})
	if err != nil {
		return hierr.Errorf(
			err, "can't remove existing hash table",
		)
	}

	docs := []interface{}{}
	for _, hash := range table {
		docs = append(docs, bson.M{
			"token": token,
			"hash":  hash,
		})
	}

	err = db.shadows.Insert(docs...)
	if err != nil {
		return hierr.Errorf(
			err, "can't insert table hash to database",
		)
	}

	return nil
}

func (db *mongodb) IsHashExists(token string, hash string) (bool, error) {
	var doc map[string]interface{}
	err := db.shadows.Find(bson.M{"token": token, "hash": hash}).One(&doc)
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (db *mongodb) GetHash(token string, number int64) (string, error) {
	var doc map[string]interface{}
	err := db.shadows.Find(
		bson.M{"token": token},
	).Skip(int(number - int64(1))).Limit(1).One(&doc)
	if err != nil {
		if err == mgo.ErrNotFound {
			return "", ErrNotFound
		}

		return "", err
	}

	return doc["hash"].(string), nil
}

func (db *mongodb) IsRecentClient(identifier string) (bool, error) {
	var doc map[string]interface{}
	err := db.clients.Find(bson.M{"client": identifier}).One(&doc)
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (db *mongodb) AddRecentClient(identifier string) error {
	err := db.clients.Insert(
		bson.M{"client": identifier, "create_date": time.Now().Unix()},
	)
	if err != nil {
		return hierr.Errorf(
			err, "can't add recent client to database",
		)
	}

	return nil
}

func (db *mongodb) GetTableSize(token string) (int64, error) {
	count, err := db.shadows.Find(bson.M{"token": token}).Count()
	if err != nil {
		return 0, hierr.Errorf(
			err, "can't obtain table size from database",
		)
	}

	if count == 0 {
		return 0, ErrNotFound
	}

	return int64(count), nil
}

func (db *mongodb) GetTokens(prefix string) ([]string, error) {
	var docs []string
	err := db.shadows.Find(
		bson.M{
			"token": bson.M{"$regex": "^" + regexp.QuoteMeta(prefix) + ".*"},
		},
	).Select(bson.M{"token": 1}).Distinct("token", &docs)
	if err != nil {
		return nil, hierr.Errorf(
			err, "can't obtain tokens from database",
		)
	}

	for i, doc := range docs {
		docs[i] = strings.TrimPrefix(doc, prefix)
	}

	sort.Strings(docs)

	return docs, nil
}

func (db *mongodb) Init() error {
	err := db.connect()
	if err != nil {
		return hierr.Errorf(
			err, "can't establish database connection",
		)
	}

	go func() {
		for range time.Tick(time.Minute) {
			db.cleanupRecentClients()
		}
	}()

	go func() {
		for range time.Tick(time.Second * 5) {
			db.ensureConnection()
		}
	}()

	return nil
}

func (db *mongodb) connect() error {
	session, err := mgo.Dial(db.dsn)
	if err != nil {
		return err
	}

	db.session = session

	db.database = db.session.DB("")
	db.shadows = db.database.C("shadows")
	db.keys = db.database.C("keys")
	db.clients = db.database.C("clients")

	return nil
}

func (db *mongodb) ensureConnection() {
	err := db.session.Ping()
	if err == nil {
		return
	}

	log.Println(
		"database connection has gone away, " +
			"trying to reestablish database connection...",
	)

	err = db.connect()
	if err != nil {
		log.Printf("can't establish database connection: %s", err)
		return
	}

	log.Println("database connection established")
}

func (db *mongodb) cleanupRecentClients() {
	_, err := db.clients.RemoveAll(
		bson.M{
			"create_date": bson.M{
				"$lt": time.Now().Unix() - int64(db.hashTTL),
			},
		},
	)
	if err != nil {
		log.Println(
			hierr.Errorf(err, "can't cleanup recent clients"),
		)
	}
}
