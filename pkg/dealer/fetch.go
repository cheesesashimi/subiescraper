package dealer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
)

const (
	dealersByStatePath string = "https://www.subaru.com/services/dealers/by/state"
	newCarPath         string = "/apis/widget/INVENTORY_LISTING_DEFAULT_AUTO_NEW:inventory-data-bus1/getInventory"
	usedCarPath        string = "/apis/widget/INVENTORY_LISTING_DEFAULT_AUTO_USED:inventory-data-bus1/getInventory"
)

func GetDealerAndInventory(d DealerResponse) (Dealer, error) {
	out := Dealer{}
	out.Dealer = d

	var wg sync.WaitGroup
	wg.Add(2)

	var newErr error
	var newInventory InventoryResponse

	var usedErr error
	var usedInventory InventoryResponse

	go func() {
		defer wg.Done()
		newInventory, newErr = getInventoryFromPath(d, newCarPath)
	}()

	go func() {
		defer wg.Done()
		usedInventory, usedErr = getInventoryFromPath(d, usedCarPath)
	}()

	wg.Wait()

	if newErr != nil {
		return out, fmt.Errorf("could not get new inventory from %s: %w", d.Name, newErr)
	}

	if usedErr != nil {
		return out, fmt.Errorf("could not get used inventory from %s: %w", d.Name, usedErr)
	}

	out.New = newInventory
	out.Used = usedInventory

	return out, nil
}

func getInventoryFromPath(d DealerResponse, inventoryPath string) (InventoryResponse, error) {
	out := InventoryResponse{}

	u, err := url.Parse(d.SiteURL)
	if err != nil {
		return out, fmt.Errorf("could not parse url: %w", err)
	}

	u.Path = inventoryPath
	q := url.Values{
		"make": []string{
			"Subaru",
			"Toyota",
		},
		"model": []string{
			"86",
			"BRZ",
			"WRX",
		},
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return out, err
	}

	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return out, err
	}

	if err := json.Unmarshal(respBytes, &out); err != nil {
		return out, err
	}

	for i, item := range out.Inventory {
		out.Inventory[i].Link = getDirectLink(resp.Request.URL, item.Link)
	}

	for i, item := range out.PageInfo.TrackingData {
		out.PageInfo.TrackingData[i].Link = getDirectLink(resp.Request.URL, item.Link)
	}

	return out, nil
}

func ByState(state string) chan DealerStream {
	dealerStream := make(chan DealerStream)

	go func() {
		for dealerResp := range GetDealersByStateWithRedirects(state) {
			if dealerResp.Err != nil {
				continue
			}

			d, err := GetDealerAndInventory(dealerResp.DealerResponse)
			dealerStream <- DealerStream{
				Dealer: d,
				Err:    err,
			}
		}

		close(dealerStream)
	}()

	return dealerStream
}

func GetDealersByState(state string) ([]DealerResponse, error) {
	out := []DealerResponse{}

	u, err := url.Parse(dealersByStatePath)
	q := u.Query()
	q.Add("state", state)
	q.Add("type", "Active")
	u.RawQuery = q.Encode()
	if err != nil {
		return out, fmt.Errorf("could not parse url: %w", err)
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return out, fmt.Errorf("could not retrieve dealers in %s: %w", state, err)
	}

	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return out, fmt.Errorf("could not read response: %w", err)
	}

	if err := json.Unmarshal(respBytes, &out); err != nil {
		return out, fmt.Errorf("could not parse dealer response: %w", err)
	}

	for i, d := range out {
		siteURL, err := getDealerHostnameRedirect(d)
		if err != nil {
			continue
		}

		out[i].SiteURL = siteURL
	}

	return out, nil
}

func GetDealersByStateWithRedirects(state string) chan DealerResponseStream {
	dealerRespChan := make(chan DealerResponseStream)

	go func() {
		dealerResps, err := GetDealersByState(state)
		if err != nil {
			dealerRespChan <- DealerResponseStream{
				Err: err,
			}
			close(dealerRespChan)
			return
		}

		for _, dealerResp := range dealerResps {
			siteURL, err := getDealerHostnameRedirect(dealerResp)
			if err == nil {
				dealerResp.SiteURL = siteURL
			}

			dealerRespChan <- DealerResponseStream{
				DealerResponse: dealerResp,
				Err:            err,
			}
		}

		close(dealerRespChan)
	}()

	return dealerRespChan
}

func getDirectLink(respURL *url.URL, link string) string {
	u := url.URL{
		Scheme: respURL.Scheme,
		Host:   respURL.Host,
		Path:   link,
	}

	return u.String()
}

func getDealerHostnameRedirect(d DealerResponse) (string, error) {
	siteURL := ""

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			siteURL = req.URL.String()
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(d.SiteURL)
	if err != nil {
		return "", fmt.Errorf("could not get dealer hostname redirect: %w", err)
	}

	defer resp.Body.Close()

	return siteURL, nil
}
