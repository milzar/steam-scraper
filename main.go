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

	filterGames()
	//processStoreEntry(StoreEntryDTO{
	//	ID: 204450,
	//})
	//getReviews("570")
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

func getReviews(gameId string) {
	defer timeTrack(time.Now(), "getReviews")

	set := make(map[string]bool)

	start := time.Now()
	reviewCount := 0

	steamUrl := getGameUrl(gameId, "*")

	res, err := http.Get(steamUrl.String())
	if err != nil {
		log.Println(err)
	}
	set["*"] = true
	gameResponse := GameResponse{}
	parseResponse(res, &gameResponse)
	reviewCount += len(gameResponse.Reviews)

	for !set[gameResponse.Cursor] {
		set[gameResponse.Cursor] = true

		steamUrl = getGameUrl(gameId, gameResponse.Cursor)

		log.Println("Url is " + steamUrl.String())

		res, err := http.Get(steamUrl.String())
		if err != nil {
			log.Println(err)
		}
		gameResponse = GameResponse{}
		parseResponse(res, gameResponse)
		reviewCount += len(gameResponse.Reviews)
	}
	end := time.Now()
	log.Printf("Total no of reviews: %v in %v s", strconv.Itoa(reviewCount), end.Unix()-start.Unix())
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
