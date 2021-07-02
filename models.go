package main

type ReviewAuthor struct {
	SteamId string `json:"steamid"`
}

type GameReview struct {
	Author ReviewAuthor `json:"author"`
	Review string       `json:"review"`
}

type ReviewQuerySummary struct {
	TotalReviews int `json:"total_reviews"`
}

type GameResponse struct {
	Reviews      []GameReview       `json:"reviews"`
	Cursor       string             `json:"cursor"`
	QuerySummary ReviewQuerySummary `json:"query_summary"`
}

type StoreEntriesResponse struct {
	AppList StoreEntriesList `json:"applist"`
}

type StoreEntriesList struct {
	Apps []StoreEntry `json:"apps"`
}

type StoreEntry struct {
	AppId int    `json:"appid"`
	Name  string `json:"name"`
}

type EntryDetailsData struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type EntryDetails struct {
	Data EntryDetailsData `json:"data"`
}

type EntryDetailsResponse map[string]EntryDetails

type GameSimilarity struct {
	GameId int
	Count  int
}
