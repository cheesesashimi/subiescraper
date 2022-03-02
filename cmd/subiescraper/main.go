package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cheesesashimi/subiescraper/pkg/dealer"
	"github.com/cheesesashimi/subiescraper/pkg/html"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "subiescraper",
		Usage: "Scrape Subaru dealer inventory in North America",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "state",
				Usage:       "What states to scrape, can be combined: --state PA --state OH",
				DefaultText: "PA",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Write output to JSON file by state (data-<state>.json)",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "html",
				Usage: "Generate an HTML report by state (data-<state>.html)",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {
			return queryDealers(c.StringSlice("state"), c.Bool("json"), c.Bool("html"))
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
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

func renderToDisk(dealers []dealer.Dealer, state string) error {
	filename := fmt.Sprintf("index-%s.html", strings.ToLower(state))
	fmt.Println("Rendering to", filename)
	page := html.DealersPage(dealers)
	return ioutil.WriteFile(filename, []byte(page), 0755)
}

func jsonToDisk(dealers []dealer.Dealer, state string) error {
	filename := fmt.Sprintf("data-%s.json", strings.ToLower(state))
	fmt.Println("Dumping JSON to:", filename)

	outBytes, err := json.Marshal(dealers)
	if err != nil {
		return fmt.Errorf("could not marshal to JSON: %w", err)
	}

	return ioutil.WriteFile(filename, outBytes, 0755)
}

func queryDealers(states []string, toJSON, toHTML bool) error {
	fmt.Println("Will query for Subarus in:", strings.Join(states, ", "))

	if toJSON {
		fmt.Println("Will write results to JSON files")
	}

	if toHTML {
		fmt.Println("Will write results to HTML files")
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

		if toHTML {
			if err := renderToDisk(dealers, state); err != nil {
				return fmt.Errorf("could not write dealer HTML to disk: %w", err)
			}
		}

		if toJSON {
			if err := jsonToDisk(dealers, state); err != nil {
				return fmt.Errorf("could not write dealer JSON to disk: %w", err)
			}
		}
	}

	return nil
}
