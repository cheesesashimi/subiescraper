package main

import (
	"fmt"
)

func main() {
	subaruQuerier := NewSubaruQuery()

	states := []string{
		"PA",
		"OH",
		"NJ",
	}

	/*
		for _, state := range states {
			for dealer := range dealer.GetDealersByStateWithRedirects(state) {
				fmt.Println(dealer.SiteURL)
			}
		}
	*/

	for _, state := range states {
		for dealer := range subaruQuerier.ByState(state) {
			if dealer.Err != nil {
				fmt.Println("could not retrieve dealer info, skipping!")
				continue
			}
			fmt.Println(dealer.SiteURL)
		}
	}
}
