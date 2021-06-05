package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const SteamBaseUrl = "https://store.steampowered.com/appreviews/"

//csgo 730
//siege 359550
//dota 2 570
func main() {
	fmt.Println("Starting")

	getReviews("570")
}

func getReviews(gameId string) {
	set := make(map[string]bool)

	start := time.Now()
	reviewCount := 0

	steamUrl := getGameUrl(gameId, "*")

	res, err := http.Get(steamUrl.String())
	if err != nil {
		fmt.Println(err)
	}
	set["*"] = true
	gameResponse := parseResponse(res)
	reviewCount += len(gameResponse.Reviews)

	for !set[gameResponse.Cursor] {
		set[gameResponse.Cursor] = true

		steamUrl = getGameUrl(gameId, gameResponse.Cursor)

		fmt.Println("Url is " + steamUrl.String())

		res, err := http.Get(steamUrl.String())
		if err != nil {
			fmt.Println(err)
		}
		gameResponse = parseResponse(res)
		reviewCount += len(gameResponse.Reviews)
	}
	end := time.Now()
	fmt.Printf("Total no of reviews: %v in %v s", strconv.Itoa(reviewCount), end.Unix()-start.Unix())
}

func parseResponse(res *http.Response) GameResponse {
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(string(bodyBytes))

	gr := GameResponse{}
	err = json.Unmarshal(bodyBytes, &gr)

	//fmt.Printf("%+v\n", gr)

	return gr
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
