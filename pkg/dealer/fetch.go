package dealer

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheesesashimi/subiescraper/pkg/utils"
)

const (
	dealersByStatePath string = "https://www.subaru.com/services/dealers/by/state"
	newCarPath         string = "/apis/widget/INVENTORY_LISTING_DEFAULT_AUTO_NEW:inventory-data-bus1/getInventory"
	usedCarPath        string = "/apis/widget/INVENTORY_LISTING_DEFAULT_AUTO_USED:inventory-data-bus1/getInventory"
)

func FromDisk(filename string) (map[string][]Dealer, error) {
	out := map[string][]Dealer{}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return out, err
	}

	err = json.Unmarshal(b, &out)

	return out, err
}

func GetDealerResponseFromReader(r io.Reader, hostname string) (DealerResponse, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	name := ""
	address := ""
	extendedAddress := ""
	locality := ""
	state := ""
	zip := ""

	doc.Find(".vcard").Each(func(i int, s *goquery.Selection) {
		if name == "" {
			name = strings.TrimSpace(s.Find(".org").Text())
		}

		if address == "" {
			address = strings.TrimSpace(s.Find(".street-address").Text())
		}

		if extendedAddress == "" {
			extendedAddress = strings.TrimSpace(s.Find(".extended-address").Text())
		}

		if locality == "" {
			locality = strings.TrimSpace(s.Find(".locality").Text())
		}

		if state == "" {
			state = strings.TrimSpace(s.Find(".region").Text())
		}

		if zip == "" {
			zip = strings.TrimSpace(s.Find(".postal-code").Text())
		}
	})

	return DealerResponse{
		Name:    name,
		SiteURL: utils.HostnameToURL(hostname),
		Address: Address{
			Street:  address,
			Street2: extendedAddress,
			City:    locality,
			State:   state,
			Zipcode: zip,
		},
	}, nil
}

func GetDealerResponseFromLandingPage(link string) (DealerResponse, error) {
	out := DealerResponse{}

	client := &http.Client{}

	resp, err := client.Get(link)
	if err != nil {
		return out, err
	}

	defer resp.Body.Close()

	return GetDealerResponseFromReader(resp.Body, resp.Request.URL.Host)
}

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
			siteURL, dnsNames, err := getDealerHostnameRedirect(dealerResp)
			if err == nil {
				dealerResp.SiteURL = siteURL
			}

			dealerRespChan <- DealerResponseStream{
				DealerResponse: dealerResp,
				DNSNames:       dnsNames,
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

func GetDealerHostnameRedirect(d DealerResponse) (string, []string, error) {
	return getDealerHostnameRedirect(d)
}

func getDealerHostnameRedirect(d DealerResponse) (string, []string, error) {
	client := &http.Client{}

	resp, err := client.Get(d.SiteURL)
	if err != nil {
		return "", []string{}, err
	}

	defer resp.Body.Close()

	dnsNames := []string{}

	if resp.TLS != nil {
		for _, cert := range resp.TLS.PeerCertificates {
			if cert != nil {
				dnsNames = append(dnsNames, cert.DNSNames...)
			}
		}
	}

	return resp.Request.URL.String(), dnsNames, err
}
