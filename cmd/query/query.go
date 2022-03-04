package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/cheesesashimi/subiescraper/pkg/dealer"
)

const (
	dealersByStatePath string = "https://www.subaru.com/services/dealers/by/state"
	newCarPath         string = "/apis/widget/INVENTORY_LISTING_DEFAULT_AUTO_NEW:inventory-data-bus1/getInventory"
	usedCarPath        string = "/apis/widget/INVENTORY_LISTING_DEFAULT_AUTO_USED:inventory-data-bus1/getInventory"
)

type SubaruQuerier interface {
	ByState(string) chan dealer.DealerResponseStream
}

type subaruQuery struct{}

func NewSubaruQuery() SubaruQuerier {
	return subaruQuery{}
}

func (s subaruQuery) ByState(state string) chan dealer.DealerResponseStream {
	dealerRespChan := make(chan dealer.DealerResponseStream)

	go func() {
		dealerResps, err := s.getDealersByState(state)
		if err != nil {
			dealerRespChan <- dealer.DealerResponseStream{
				Err: err,
			}
			close(dealerRespChan)
			return
		}

		for _, dealerResp := range dealerResps {
			// This is needed because the dealer URLs that subaru.com returns are
			// redirect URLs owned by Subaru, e.g.:
			// https://bowsersubaru.dealer.subaru.com
			//
			// What we want is:
			// https://www.bowsersubaru.com
			//
			// PS: Don't do business with this particular dealer :P
			results, err := getDealerHostnameRedirect(dealerResp)
			if err == nil {
				dealerResp.SiteURL = results.url.String()
			}

			dealerRespChan <- dealer.DealerResponseStream{
				DealerResponse: dealerResp,
				DNSNames:       results.dnsNames,
				Err:            err,
			}
		}

		close(dealerRespChan)
	}()

	return dealerRespChan
}

func (s subaruQuery) getDealersByState(state string) ([]dealer.DealerResponse, error) {
	out := []dealer.DealerResponse{}

	u, err := url.Parse(dealersByStatePath)
	q := u.Query()
	q.Add("state", state)
	q.Add("type", "Active")
	u.RawQuery = q.Encode()
	if err != nil {
		return out, fmt.Errorf("could not parse url: %w", err)
	}

	// curl https://www.subaru.com/services/dealers/by/state?state=<state>&type=Active
	resp, err := getHTTPRequest(u)
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

type hostRedirect struct {
	url                *url.URL
	dnsNames           []string
	landingPageContent []byte
}

func getDealerHostnameRedirect(d dealer.DealerResponse) (*hostRedirect, error) {
	// This does three things:
	// 1. We make an HTTP request to the URL provided by Subaru and follow the
	// redirect to get the dealers actual URL.
	// 2. We scrape the certificate for all the DNS names that this certificate
	// is valid for. This is because we can locate additional dealers based upon
	// that info.
	// 3. While not useful for the Subaru-specific case (since we already have
	// their dealership info), we grab the landing page contents.
	results := &hostRedirect{}

	u, err := url.Parse(d.SiteURL)
	if err != nil {
		return nil, err
	}

	resp, err := getHTTPRequest(u)
	if err != nil {
		return nil, err
	}

	results.url = resp.Request.URL

	defer resp.Body.Close()

	pageBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results.landingPageContent = pageBytes
	results.dnsNames = []string{}

	if resp.TLS != nil {
		for _, cert := range resp.TLS.PeerCertificates {
			if cert != nil {
				results.dnsNames = append(results.dnsNames, cert.DNSNames...)
			}
		}
	}

	return results, nil
}

func getHTTPRequest(u *url.URL) (*http.Response, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36")

	return http.DefaultClient.Do(req)
}
