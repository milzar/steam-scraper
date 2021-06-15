package main

type StoreEntryDTO struct {
	ID   int    `bson:"_id,omitempty"`
	Name string `bson:"title,omitempty"`
}

type GameReviewDTO struct {
	AppId   int      `bson:"_id,omitempty"`
	Users   []string `bson:"users,omitempty"`
	Reviews []string `bson:"reviews,omitempty"`
}
