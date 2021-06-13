package main

type StoreEntryDTO struct {
	ID   int    `bson:"_id,omitempty"`
	Name string `bson:"title,omitempty"`
}
