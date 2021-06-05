package main

type ReviewAuthor struct {
	SteamId string `json:"steamid"`
}

type GameReview struct {
	Author ReviewAuthor `json:"author"`
}

type GameResponse struct {
	Reviews []GameReview `json:"reviews"`
	Cursor  string       `json:"cursor"`
}
