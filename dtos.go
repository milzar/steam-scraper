package main

type Game struct {
	ID   int    `bson:"_id,omitempty"`
	Name string `bson:"title,omitempty"`
}
