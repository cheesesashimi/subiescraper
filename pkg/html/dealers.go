package html

import (
	"fmt"
	"sort"

	"github.com/cheesesashimi/subiescraper/pkg/dealer"
	"github.com/julvo/htmlgo"
	a "github.com/julvo/htmlgo/attributes"
)

func DealersPage(dealers []dealer.Dealer) htmlgo.HTML {
	out := []htmlgo.HTML{}

	for _, d := range dealers {
		var newCars []htmlgo.HTML
		var usedCars []htmlgo.HTML

		if len(d.New.PageInfo.TrackingData) != 0 {
			newCars = []htmlgo.HTML{
				htmlgo.H3_("New Cars:"),
				getInventoryTable(d.New),
			}
		} else {
			newCars = []htmlgo.HTML{htmlgo.H3_("No new cars")}
		}

		if len(d.Used.PageInfo.TrackingData) != 0 {
			usedCars = []htmlgo.HTML{
				htmlgo.H3_("Used Cars:"),
				getInventoryTable(d.Used),
			}
		} else {
			usedCars = []htmlgo.HTML{htmlgo.H3_("No used cars")}
		}

		c := htmlgo.Div_(
			htmlgo.Div_(
				htmlgo.H2_(htmlgo.Text(d.Dealer.Name)),
			),
			htmlgo.Div_(newCars...),
			htmlgo.Div_(usedCars...),
		)

		out = append(out, c)
	}

	return htmlgo.Html5_(
		htmlgo.Head_(),
		htmlgo.Body_(out...),
	)
}

func getInventoryTable(inventory dealer.InventoryResponse) htmlgo.HTML {
	sort.Slice(inventory.PageInfo.TrackingData, func(i, j int) bool {
		return inventory.PageInfo.TrackingData[i].ModelYear > inventory.PageInfo.TrackingData[j].ModelYear
	})

	listItems := []htmlgo.HTML{}
	for _, item := range inventory.PageInfo.TrackingData {
		carLine := fmt.Sprintf("%d %s %s %s (%s)", item.ModelYear, item.Make, item.Model, item.Trim, item.ExteriorColor)
		listItems = append(listItems, htmlgo.Li_(htmlgo.A([]a.Attribute{a.Href_(item.Link)}, htmlgo.Text(carLine))))
	}

	return htmlgo.Ul_(listItems...)
}
