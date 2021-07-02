package main

type StoreEntryDTO struct {
	ID   int    `bson:"_id,omitempty"`
	Name string `bson:"title,omitempty"`
}

type GameReviewDTO struct {
	AppId int      `bson:"_id,omitempty"`
	Users []string `bson:"users,omitempty"`
	//Reviews []string `bson:"reviews,omitempty"`
}

type UserLinkDTO struct {
	UserId        string `bson:"_id,omitempty"`
	GamesReviewed []int  `bson:"gamesReviewed,omitempty"`
}

type GameLinkDTO struct {
	GameId       int              `bson:"_id,omitempty"`
	SimilarGames []GameSimilarity `bson:"similarGames,omitempty"`
}
