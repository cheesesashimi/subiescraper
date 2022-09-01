package dealer

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheesesashimi/subiescraper/pkg/utils"
	aggError "k8s.io/apimachinery/pkg/util/errors"
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

	fields := []string{
		"address1",
		"address2",
		"city",
		"country",
		"postalCode",
		"stateProvince",
		"dealershipName",
	}

	extracted := map[string]string{}

	foundText := ""

	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		foundText = s.Text()
		foundIt := strings.Contains(foundText, "DDC.dataLayer['dealership'] = {")
		return !foundIt
	})

	lines := strings.Split(foundText, "\n")
	for _, line := range lines {
		for _, field := range fields {
			prefix := fmt.Sprintf("\"%s\": ", field)
			if strings.HasPrefix(line, prefix) {
				line = strings.ReplaceAll(line, prefix, "")
				line = strings.TrimRight(line, ",")
				unquoted, err := strconv.Unquote(line)
				if err != nil {
					panic(err)
				}
				extracted[field] = unquoted
			}
		}
	}

	dr := DealerResponse{
		Name:    extracted["dealershipName"],
		SiteURL: utils.HostnameToURL(hostname),
		Address: Address{
			Street:  extracted["address1"],
			Street2: extracted["address2"],
			City:    extracted["city"],
			State:   extracted["stateProvince"],
			Zipcode: extracted["postalCode"],
		},
	}

	fmt.Println(dr)

	return dr, nil
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

func GetDealerAndInventoryFromLink(link string, inventoryQuery url.Values) (Dealer, error) {
	dr, err := GetDealerResponseFromLandingPage(link)
	if err != nil {
		return Dealer{}, err
	}

	return GetDealerAndInventory(dr, inventoryQuery)
}

func GetDealerAndInventory(d DealerResponse, inventoryQuery url.Values) (Dealer, error) {
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
		newInventory, newErr = getInventoryFromPath(d, newCarPath, inventoryQuery)
	}()

	go func() {
		defer wg.Done()
		usedInventory, usedErr = getInventoryFromPath(d, usedCarPath, inventoryQuery)
	}()

	wg.Wait()

	errs := []error{}

	if newErr != nil {
		errs = append(errs, fmt.Errorf("could not get new inventory: %w", newErr))
	}

	if usedErr != nil {
		errs = append(errs, fmt.Errorf("could not get used inventory: %w", usedErr))
	}

	out.New = newInventory
	out.Used = usedInventory

	return out, aggError.NewAggregate(errs)
}

func getInventoryFromPath(d DealerResponse, inventoryPath string, inventoryQuery url.Values) (InventoryResponse, error) {
	out := InventoryResponse{}

	u, err := url.Parse(d.SiteURL)
	if err != nil {
		return out, fmt.Errorf("could not parse url: %w", err)
	}

	u.Path = inventoryPath
	u.RawQuery = inventoryQuery.Encode()

	client := &http.Client{}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return out, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return out, err
	}

	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("could not retrieve cars for %s (%s): HTTP %d - %s", d.Name, u.String(), resp.StatusCode, http.StatusText(resp.StatusCode))
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

			d, err := GetDealerAndInventory(dealerResp.DealerResponse, url.Values{
				"make":  []string{"Subaru"},
				"model": []string{"WRX", "BRZ", "Outback"},
			})
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
