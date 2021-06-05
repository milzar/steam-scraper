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
	game := Game{
		ID:   entry.AppId,
		Name: entry.Name,
	}
	gamesCollection := d.db.Collection(gamesCollectionName)

	_, err := gamesCollection.InsertOne(context.TODO(), game)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *DataBase) findGame(id int) bool {
	gamesCollection := d.db.Collection(gamesCollectionName)

	return nil == gamesCollection.FindOne(context.TODO(), bson.M{"_id": id}).Err()
}
