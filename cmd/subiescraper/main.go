package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/cheesesashimi/subiescraper/pkg/dealer"
	"github.com/cheesesashimi/subiescraper/pkg/html"
)

func alts() {
	alts, err := ioutil.ReadFile("alternates.txt")
	if err != nil {
		log.Fatal(err)
	}

	for _, alt := range strings.Split(string(alts), "\n") {
		if alt != "" && !strings.Contains(alt, "dealer.com") {
			link := "https://www." + alt

			time.Sleep(500 * time.Millisecond)
			fmt.Println(link)

			d, err := dealer.GetDealerResponseFromLandingPage(link)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(d)
		}
	}
}

func main() {
	all()
}

func printCarDetail(td []dealer.TrackingData, carType string) {
	if len(td) == 0 {
		fmt.Println("No", carType, "cars")
		return
	}

	fmt.Println(strings.Title(carType), "Cars:")
	for _, item := range td {
		fmt.Printf("- %d %s %s %s (%s) - %s\n", item.ModelYear, item.Make, item.Model, item.Trim, item.ExteriorColor, item.Link)
	}
}

func printDealerDetail(d dealer.Dealer) {
	fmt.Println("Dealer:", d.Dealer.Name, d.Dealer.SiteURL)
	printCarDetail(d.New.PageInfo.TrackingData, "new")
	printCarDetail(d.Used.PageInfo.TrackingData, "used")
}

func renderToDisk(dealers []dealer.Dealer, state string) {
	filename := fmt.Sprintf("index-%s.html", strings.ToLower(state))
	fmt.Println("Rendering to", filename)
	page := html.DealersPage(dealers)
	if err := ioutil.WriteFile(filename, []byte(page), 0755); err != nil {
		panic(err)
	}
}

func jsonToDisk(dealers []dealer.Dealer, state string) {
	filename := fmt.Sprintf("data-%s.json", strings.ToLower(state))
	fmt.Println("Dumping JSON to:", filename)

	outBytes, err := json.Marshal(dealers)
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(filename, outBytes, 0755); err != nil {
		panic(err)
	}
}

func all() {
	states := []string{
		"CT",
		"DE",
		"MD",
		"NJ",
		"NY",
		"OH",
		"PA",
		"VA",
		"WV",
	}

	for _, state := range states {
		fmt.Println("Getting dealers in", state)
		dealers := []dealer.Dealer{}
		for d := range dealer.ByState(state) {
			if d.Err != nil {
				continue
			}
			printDealerDetail(d.Dealer)
			dealers = append(dealers, d.Dealer)
		}
		renderToDisk(dealers, state)
		jsonToDisk(dealers, state)
	}
}
