package main

import (
	"encoding/json"
	"golang.org/x/net/context"
	"googlemaps.github.io/maps"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type store struct {
	PlaceID       string                       `json:"google_place_id"`
	Name          string                       `json:"name"`
	Lat           float64                      `json:"latitude"`
	Lng           float64                      `json:"longitude"`
	PhoneNumber   string                       `json:"phone"`
	Address       string                       `json:"address"`
	Country       string                       `json:"country"`
	State         string                       `json:"state"`
	City          string                       `json:"city"`
	AvailableTime map[time.Weekday]interface{} `json:"available_time"`
}

func search(c *maps.Client, queryString *string, nextPageToken *string, language *string) (maps.PlacesSearchResponse, error) {
	r := &maps.TextSearchRequest{
		Query:     *queryString,
		PageToken: *nextPageToken,
		Language:  *language,
	}

	return c.TextSearch(context.Background(), r)
}

func fetchStore(c *maps.Client, placeId *string, language *string) store {
	r := &maps.PlaceDetailsRequest{
		PlaceID:  *placeId,
		Language: *language,
	}

	googleStore, err := c.PlaceDetails(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	store := store{}

	store.PlaceID = googleStore.PlaceID
	store.Name = googleStore.Name
	store.Lat = googleStore.Geometry.Location.Lat
	store.Lng = googleStore.Geometry.Location.Lng
	store.PhoneNumber = googleStore.InternationalPhoneNumber
	store.Address = googleStore.FormattedAddress

	if googleStore.OpeningHours != nil && len(googleStore.OpeningHours.Periods) > 0 {

		store.AvailableTime = make(map[time.Weekday]interface{})
		for _, p := range googleStore.OpeningHours.Periods {
			day := p.Open.Day
			if day == 0 {
				day = 7
			}
			store.AvailableTime[day] = []string{p.Open.Time, p.Close.Time}
		}
	}

	for _, addressComponent := range googleStore.AddressComponents {
		for _, addressType := range addressComponent.Types {
			switch addressType {
			case "country":
				store.Country = addressComponent.ShortName
			case "administrative_area_level_1":
				store.State = addressComponent.ShortName
			case "locality":
				store.City = addressComponent.ShortName
			}
		}
	}

	return store
}

func main() {
	arg_num := len(os.Args)

	if arg_num < 4 {
		log.Fatalf(os.Args[0] + " store_name city_name language(en|ja|zh-TW)")
	}

	key := os.Getenv("GOOGLE_KEY")

	if key == "" {
		log.Fatalf("No google api key, please set  env GOOGLE_KEY variable")
	}

	storename := os.Args[1]
	city := os.Args[2]
	language := os.Args[3]
	queryString := storename + " in " + city
	NextPageToken := ""
	storeCollect := []store{}

	c, err := maps.NewClient(maps.WithAPIKey(key))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	for {
		places, err := search(c, &queryString, &NextPageToken, &language)
		if err != nil {
			log.Fatalf("fatal error: %s", err)
		}

		for _, place := range places.Results {
			store := fetchStore(c, &place.PlaceID, &language)
			storeCollect = append(storeCollect, store)
		}

		if NextPageToken = places.NextPageToken; NextPageToken == "" {
			break
		}
	}

	storeJson, err := json.Marshal(storeCollect)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	err = ioutil.WriteFile(storename+"_in_"+city+".json", storeJson, 0644)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

}
