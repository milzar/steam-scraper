package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var savedGames int32

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
}

func main() {
	storeEntries := fetchStoreEntries()

	saveStoreEntries(storeEntries)

	//getReviews("570")
}

func saveStoreEntries(entries []StoreEntry) {
	defer timeTrack(time.Now(), "saveStoreEntries")

	fmt.Printf("Saving %v entries into database\n", len(entries))

	var wg sync.WaitGroup

	for _, entry := range entries {
		wg.Add(1)
		go saveEntry(entry, &wg)
	}
	wg.Wait()
}

func saveEntry(entry StoreEntry, wg *sync.WaitGroup) {
	defer wg.Done()
	//if database.findGame(entry.AppId) {
	//	fmt.Println("Already found " + entry.Name)
	//	return
	//}
	//fmt.Printf("Saving %s\n", entry.Name)

	setMe(getMe() + 1)
	fmt.Printf("Saved %v games\n", getMe())
	database.saveStoreEntry(entry)

	//res, err := http.Get(steamEntryDetailsUrl + strconv.Itoa(entry.AppId))
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//entryDetailsResponse := EntryDetailsResponse{}
	//
	//parseResponse(res, &entryDetailsResponse)

	//if entryDetailsResponse[strconv.Itoa(entry.AppId)].Data.Type == "game" {
	//	fmt.Printf("Saving %s\n", entry.Name)
	//	database.saveStoreEntry(entry)
	//}
}

func fetchStoreEntries() []StoreEntry {
	defer timeTrack(time.Now(), "fetchStoreEntries")

	fmt.Println("Fetching items from steam store")

	res, err := http.Get(steamStoreEntriesUrl)
	if err != nil {
		log.Fatal(err)
	}
	steamEntriesResponse := StoreEntriesResponse{}
	parseResponse(res, &steamEntriesResponse)
	fmt.Printf("Found %v entries\n", len(steamEntriesResponse.AppList.Apps))
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
		fmt.Println(err)
	}
	set["*"] = true
	gameResponse := GameResponse{}
	parseResponse(res, &gameResponse)
	reviewCount += len(gameResponse.Reviews)

	for !set[gameResponse.Cursor] {
		set[gameResponse.Cursor] = true

		steamUrl = getGameUrl(gameId, gameResponse.Cursor)

		fmt.Println("Url is " + steamUrl.String())

		res, err := http.Get(steamUrl.String())
		if err != nil {
			fmt.Println(err)
		}
		gameResponse = GameResponse{}
		parseResponse(res, gameResponse)
		reviewCount += len(gameResponse.Reviews)
	}
	end := time.Now()
	fmt.Printf("Total no of reviews: %v in %v s", strconv.Itoa(reviewCount), end.Unix()-start.Unix())
}

func parseResponse(res *http.Response, value interface{}) {
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(string(bodyBytes))

	err = json.Unmarshal(bodyBytes, value)

	//fmt.Printf("%+v\n", gr)
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
