package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

const gamesCollectionName = "games"
const storeEntriesCollection = "store-entries"

type DataBase struct {
	db *mongo.Database
}

func (d *DataBase) initDatabase(databaseUrl string) {

	client, err := mongo.NewClient(options.Client().ApplyURI(databaseUrl))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to db")
	d.db = client.Database("valkyrie")
}

func (d *DataBase) saveStoreEntry(entry StoreEntry) {
	game := StoreEntryDTO{
		ID:   entry.AppId,
		Name: entry.Name,
	}
	gamesCollection := d.db.Collection(storeEntriesCollection)

	_, err := gamesCollection.InsertOne(context.TODO(), game)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *DataBase) findStoreEntry(id int) bool {
	gamesCollection := d.db.Collection(storeEntriesCollection)

	return nil == gamesCollection.FindOne(context.TODO(), bson.M{"_id": id}).Err()
}

func (d *DataBase) findStoreEntries() *mongo.Cursor {
	findOptions := options.Find()
	// Sort by `price` field descending
	findOptions.SetSort(bson.D{{"_id", 1}})

	storeEntriesCollection := d.db.Collection(storeEntriesCollection)

	res, err := storeEntriesCollection.Find(context.TODO(), bson.M{}, findOptions)

	if err != nil {
		log.Fatal(err)
	}

	return res
}

func (d *DataBase) saveGame(entry StoreEntryDTO) {
	gamesCollection := d.db.Collection(gamesCollectionName)

	_, err := gamesCollection.InsertOne(context.TODO(), entry)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *DataBase) findGames() *mongo.Cursor {
	findOptions := options.Find()
	// Sort by `price` field descending
	findOptions.SetSort(bson.D{{"_id", 1}})

	storeEntriesCollection := d.db.Collection(gamesCollectionName)

	res, err := storeEntriesCollection.Find(context.TODO(), bson.M{}, findOptions)

	if err != nil {
		log.Fatal(err)
	}

	return res

}
