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
const gameReviewsCollection = "game-reviews"
const userLinksCollection = "user-links"

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
	findOptions.SetSort(bson.D{{"_id", 1}})

	storeEntriesCollection := d.db.Collection(gamesCollectionName)

	res, err := storeEntriesCollection.Find(context.TODO(), bson.M{}, findOptions)

	if err != nil {
		log.Fatal(err)
	}

	return res

}

func (d *DataBase) findLastProcessedReview() GameReviewDTO {
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{"_id", -1}})

	gamesCollection := d.db.Collection(gameReviewsCollection)

	res := gamesCollection.FindOne(context.TODO(), bson.M{}, findOptions)

	var game GameReviewDTO
	err := res.Decode(&game)

	if err != nil {
		log.Println("Error while fetching last reviewed game")
		return GameReviewDTO{AppId: 0}
	}

	return game
}

func (d *DataBase) saveGameReview(review GameReviewDTO) {
	gamesCollection := d.db.Collection(gameReviewsCollection)

	_, err := gamesCollection.InsertOne(context.TODO(), review)
	check(err)
}

func (d *DataBase) findGameReviews() *mongo.Cursor {
	defer timeTrack(time.Now(), "findGameReviews")

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{"_id", 1}})

	gameReviewsCollection := d.db.Collection(gameReviewsCollection)

	res, err := gameReviewsCollection.Find(context.TODO(), bson.M{}, findOptions)

	if err != nil {
		log.Fatal(err)
	}

	return res
}

func (d *DataBase) findGameReview(gameId int) GameReviewDTO {
	defer timeTrack(time.Now(), "findGameReview")

	gameReviewsCollection := d.db.Collection(gameReviewsCollection)

	res := gameReviewsCollection.FindOne(context.TODO(), bson.M{"_id": gameId})

	var game GameReviewDTO
	err := res.Decode(&game)

	check(err)

	return game
}

func (d *DataBase) findUserLink(userId string) UserLinkDTO {
	userLinksCollection := d.db.Collection(userLinksCollection)

	res := userLinksCollection.FindOne(context.TODO(), bson.M{"_id": userId})

	var userLink UserLinkDTO

	if res.Err() == nil {
		err := res.Decode(&userLink)
		check(err)
	}

	return userLink
}

func (d *DataBase) updateUserLink(link UserLinkDTO) {
	updateOptions := options.Update()
	updateOptions.SetUpsert(true)

	userLinksCollection := d.db.Collection(userLinksCollection)

	update := bson.M{
		"$set": link,
	}
	_, err := userLinksCollection.UpdateOne(context.TODO(), bson.M{"_id": link.UserId}, update, updateOptions)
	check(err)
}

func (d *DataBase) findUserLinks(ids []string) []UserLinkDTO {
	userLinksCollection := d.db.Collection(userLinksCollection)

	cursor, err := userLinksCollection.Find(context.TODO(), bson.M{"_id": bson.M{"$in": ids}})
	check(err)

	var userLinks []UserLinkDTO

	err = cursor.All(context.TODO(), &userLinks)
	check(err)

	return userLinks
}
