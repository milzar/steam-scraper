package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var savedGames int32

const progressfilename = "progress.txt"

func getMe() int32 {
	return atomic.LoadInt32(&savedGames)
}

func setMe(me int32) {
	atomic.StoreInt32(&savedGames, me)
}

const SteamBaseUrl = "https://store.steampowered.com/appreviews/"
const steamEntryDetailsUrl = "https://store.steampowered.com/api/appdetails?appids="
const steamStoreEntriesUrl = "http://api.steampowered.com/ISteamApps/GetAppList/v0002/?key=STEAMKEY&format=json"

var database DataBase
var databaseUrl = os.Getenv("DATABASE_URL")

//csgo 730
//siege 359550
//dota 2 570
//dota 2 570

func init() {
	database.initDatabase(databaseUrl)

	initLogs()
}

func initLogs() {
	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func main() {
	log.Println("Starting application")
	//initStoreEntries()

	//filterGames()

	//processReviews()

	//processUserLinks()

	populateGameSimilarities()
}

func populateGameSimilarities() {
	defer timeTrack(time.Now(), "populateGameSimilarities")

	var gameReviewsList []GameReviewDTO
	cursor := database.findGameReviews()
	err := cursor.All(context.TODO(), &gameReviewsList)
	check(err)

	for _, review := range gameReviewsList {
		processGameLink(review.AppId)
	}
}

func processGameLink(gameId int) {
	log.Printf("Saving game link for %v \n", gameId)

	similarities := findSimilarGames(gameId)

	database.saveGameLink(gameId, similarities)
}

func findSimilarGames(gameId int) []GameSimilarity {
	defer timeTrack(time.Now(), "findSimilarGames")

	reviews := database.findGameReview(gameId)

	userIds := reviews.Users
	if len(userIds) == 0 {
		return []GameSimilarity{}
	}
	userLinks := database.findUserLinks(userIds)

	similarGameMap := make(map[int]int)

	for _, userLink := range userLinks {
		for _, game := range userLink.GamesReviewed {
			similarGameMap[game]++
		}
	}

	var sortedSlice []GameSimilarity
	for k, v := range similarGameMap {
		sortedSlice = append(sortedSlice, GameSimilarity{k, v})
	}

	sort.Slice(sortedSlice, func(i, j int) bool {
		return sortedSlice[i].Count > sortedSlice[j].Count
	})
	//Exclude self
	return sortedSlice[1:]
}

func processUserLinks() {
	defer timeTrack(time.Now(), "processUserLinks")

	log.Println("Processing User links")

	cursor := database.findGameReviews()

	for cursor.Next(context.TODO()) {
		var review GameReviewDTO
		err := cursor.Decode(&review)
		check(err)

		log.Printf("\n\nProcessing %v\n\n", review.AppId)

		saveUserGameLinks(review)

	}
}

func saveUserGameLinks(review GameReviewDTO) {
	defer timeTrack(time.Now(), "saveUserGameLinks")

	userIds := review.Users

	log.Printf("Processing %v users\n", len(userIds))

	userIdsMap := make(map[string]bool)
	for _, userId := range userIds {
		userIdsMap[userId] = true
	}
	log.Printf("Processing %v distinct users\n", len(userIdsMap))

	var wg sync.WaitGroup

	for userId := range userIdsMap {
		wg.Add(1)
		go saveUserGameLink(review.AppId, userId, &wg)
	}

	wg.Wait()
	log.Println()
}

func saveUserGameLink(gameId int, userId string, wg *sync.WaitGroup) {
	defer wg.Done()

	userLink := database.findUserLink(userId)

	if userLink.UserId != "" {
		alreadyExists := reviewAlreadyExists(gameId, userLink)
		if alreadyExists {
			return
		}
		userLink.GamesReviewed = append(userLink.GamesReviewed, gameId)
	} else {
		userLink.UserId = userId
		userLink.GamesReviewed = []int{gameId}
	}

	database.updateUserLink(userLink)
}

func reviewAlreadyExists(gameId int, userLink UserLinkDTO) bool {
	for _, games := range userLink.GamesReviewed {
		if games == gameId {
			return true
		}
	}
	return false
}

func processReviews() {
	var games []StoreEntryDTO

	cursor := database.findGames()
	err := cursor.All(context.TODO(), &games)
	check(err)

	lastProcessedGame := database.findLastProcessedReview()

	for i, game := range games {
		if game.ID <= lastProcessedGame.AppId {
			continue
		}
		log.Printf("Processing reviews for %v %v\n\n", game.Name, game.ID)

		gameReview, apiError := getReviews(game.ID)
		if apiError != nil {
			log.Printf("\n Error while processing reviews for %v\n", game.Name)

			time.Sleep(time.Minute * 15)
			i--
			continue
		}
		var gameSaved bool

		if len(gameReview.Users) > 100 {
			saveGameReviews(gameReview)
			gameSaved = true
		}

		log.Printf("Finished processing reviews for %v %v\n\n", game.Name, game.ID)
		log.Printf("\n %.2f percent done\n", (float32(i)/float32(len(games)))*100)

		if gameSaved {
			time.Sleep(time.Minute * 5)
		}
	}

}

func saveGameReviews(review GameReviewDTO) {
	log.Printf("\n Saving reviews \n\n")
	database.saveGameReview(review)
}

func filterGames() {
	var storeEntriesList []StoreEntryDTO
	cursor := database.findStoreEntries()
	err := cursor.All(context.TODO(), &storeEntriesList)
	check(err)

	var processedGamesList []StoreEntryDTO
	cursor = database.findGames()
	err = cursor.All(context.TODO(), &processedGamesList)
	check(err)

	processedGamesMap := make(map[int]bool)

	for _, entry := range processedGamesList {
		processedGamesMap[entry.ID] = true
	}

	log.Printf("Fetched entries from db \n")

	lastProcessedId := findLastProcessedAppId()
	log.Printf("Last processed id %v\n\n\n", lastProcessedId)

	savedGamesCount := 0

	for i := 0; i < len(storeEntriesList); i++ {
		entry := storeEntriesList[i]

		if processedGamesMap[entry.ID] {
			log.Printf("\n Already processed game %v \n\n\n", entry)
			continue
		}

		if entry.ID <= lastProcessedId {
			log.Printf("Skipping %s %v\n", entry.Name, entry.ID)
			continue
		}

		isGame, steamError := processStoreEntry(entry)

		if steamError != nil {
			const waitSeconds = 150
			log.Println("Rate limit reached")
			log.Printf("\nWaiting for %vs \n", waitSeconds)
			time.Sleep(time.Second * waitSeconds)

			i--
			continue
		}
		if isGame {
			savedGamesCount++

			log.Printf("\n\n\n Saved %v new games \n\n", savedGamesCount)
		}
		updateProgress(entry.ID)

		log.Printf("%.2f percent done\n", (float32(i)/float32(len(storeEntriesList)))*100)
	}
}

func processStoreEntry(storeEntry StoreEntryDTO) (bool, error) {
	log.Printf("Processing: %v %v ----------------\n", storeEntry.Name, storeEntry.ID)

	details, steamAPIerr := getStoreEntryDetails(storeEntry.ID)

	if steamAPIerr != nil {
		return false, steamAPIerr
	}
	time.Sleep(time.Second * 1)

	if details.Data.Type == "game" {
		log.Printf("Saving %v %v \n\n", storeEntry.Name, storeEntry.ID)
		database.saveGame(storeEntry)
		return true, nil
	}
	return false, nil

}

func findLastProcessedAppId() int {

	if !fileExists(progressfilename) {
		log.Println("Previous progress not found")
		updateProgress(0)
		return 0
	}

	data, err := ioutil.ReadFile(progressfilename)

	check(err)

	fileStr := string(data)

	digits := strings.TrimSuffix(fileStr, "\n")

	id, err := strconv.Atoi(digits)
	check(err)

	return id
}

func updateProgress(i int) {
	if !fileExists(progressfilename) {
		f, err := os.Create(progressfilename)
		check(err)

		_, err = f.WriteString("0")
		check(err)

		defer f.Close()
		return
	}

	f, err := os.OpenFile(progressfilename, os.O_WRONLY, os.ModeAppend)
	check(err)

	_, err = f.WriteString(strconv.Itoa(i))

	check(err)

	defer f.Close()
}

func initStoreEntries() {
	storeEntries := fetchStoreEntries()

	saveStoreEntries(storeEntries)
}

func saveStoreEntries(entries []StoreEntry) {
	defer timeTrack(time.Now(), "saveStoreEntries")

	log.Printf("Saving %v entries into database\n", len(entries))

	var wg sync.WaitGroup

	for _, entry := range entries {
		wg.Add(1)
		go saveEntry(entry, &wg)
	}
	wg.Wait()
}

func saveEntry(entry StoreEntry, wg *sync.WaitGroup) {
	defer wg.Done()

	setMe(getMe() + 1)
	log.Printf("Saved %v games\n", getMe())
	database.saveStoreEntry(entry)

}

func fetchStoreEntries() []StoreEntry {
	defer timeTrack(time.Now(), "fetchStoreEntries")

	log.Println("Fetching items from steam store")

	res, err := http.Get(steamStoreEntriesUrl)
	if err != nil {
		log.Fatal(err)
	}
	steamEntriesResponse := StoreEntriesResponse{}
	parseResponse(res, &steamEntriesResponse)
	log.Printf("Found %v entries\n", len(steamEntriesResponse.AppList.Apps))
	return steamEntriesResponse.AppList.Apps
}

func getReviews(gameId int) (GameReviewDTO, error) {
	defer timeTrack(time.Now(), "getReviews")
	start := time.Now()

	gameIdString := strconv.Itoa(gameId)

	var gameReviews GameReviewDTO

	gameReviews.AppId = gameId

	cursorMap := make(map[string]bool)

	steamUrl := getGameUrl(gameIdString, "*")
	log.Println("Url is " + steamUrl.String())

	res, err := http.Get(steamUrl.String())
	check(err)

	if res.StatusCode != 200 {
		log.Printf("\nAPI rate limit reached \n\n")
		return GameReviewDTO{}, errors.New("api rate limit exceeded")
	}

	cursorMap["*"] = true
	gameResponse := GameResponse{}
	parseResponse(res, &gameResponse)

	log.Printf("Game has %v reviews\n", gameResponse.QuerySummary.TotalReviews)
	const minReviewCount = 2500

	if gameResponse.QuerySummary.TotalReviews < minReviewCount {
		log.Printf("Skipping game \n")
		return gameReviews, nil
	}

	log.Printf("\n Fetched %v reviews \n", len(gameResponse.Reviews))

	gameReviews = appendReviews(gameResponse, gameReviews)

	for !cursorMap[gameResponse.Cursor] {
		cursorMap[gameResponse.Cursor] = true

		steamUrl = getGameUrl(gameIdString, gameResponse.Cursor)

		log.Println("Url is " + steamUrl.String())

		res, err := http.Get(steamUrl.String())
		check(err)
		if res.StatusCode != 200 {
			log.Printf("\nAPI rate limit reached \n\n")
			return GameReviewDTO{}, errors.New("api rate limit exceeded")
		}

		parseResponse(res, &gameResponse)

		log.Printf("\n Fetched %v reviews \n", len(gameResponse.Reviews))

		gameReviews = appendReviews(gameResponse, gameReviews)

		time.Sleep(time.Second * 3)
	}

	end := time.Now()
	log.Printf("Total no of reviews: %v in %v s", strconv.Itoa(len(gameReviews.Users)), end.Unix()-start.Unix())

	return gameReviews, nil
}

func appendReviews(gameResponse GameResponse, gameReviews GameReviewDTO) GameReviewDTO {
	for _, review := range gameResponse.Reviews {
		gameReviews.Users = append(gameReviews.Users, review.Author.SteamId)
		//gameReviews.Reviews = append(gameReviews.Reviews, review.Review)
	}
	return gameReviews
}

func parseResponse(res *http.Response, value interface{}) {
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(bodyBytes, value)

}

func getGameUrl(gameId string, cursor string) *url.URL {
	base, err := url.Parse(SteamBaseUrl)
	if err != nil {
		return nil
	}
	base.Path += gameId

	params := url.Values{}
	params.Add("json", "1")
	params.Add("filter", "all")
	params.Add("day_range", "5100")
	params.Add("purchase_type", "all")
	params.Add("cursor", cursor)
	params.Add("num_per_page", "100")
	params.Add("review_type", "all")
	params.Add("language", "all")

	base.RawQuery = params.Encode()
	return base
}

func getStoreEntryDetails(id int) (EntryDetails, error) {
	res, err := http.Get(steamEntryDetailsUrl + strconv.Itoa(id))
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode != 200 {
		log.Printf("\n\n Steam API failed \n\n\n\n")
		return EntryDetails{}, errors.New("rate limit exceeded")
	}
	entryDetailsResponse := EntryDetailsResponse{}

	parseResponse(res, &entryDetailsResponse)

	return entryDetailsResponse[strconv.Itoa(id)], nil
}
