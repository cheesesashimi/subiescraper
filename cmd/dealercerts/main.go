package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gammazero/workerpool"

	"github.com/cheesesashimi/subiescraper/pkg/dealer"
	"github.com/cheesesashimi/subiescraper/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	dealersFile           string = "dealerurls.txt"
	classifiedDealersFile string = "classified-dealers.json"
)

func getInterestedMakes() []string {
	return []string{
		"acura",
		"honda",
		"hyundai",
		"lexus",
		"nissan",
		"subaru",
		"toyota",
		"volkswagen",
		"vw",
	}
}

func classifyDealers(dealers []dealer.DealerResponse) map[string][]dealer.DealerResponse {
	knownMakes := getInterestedMakes()
	out := map[string][]dealer.DealerResponse{}

	for _, knownMake := range knownMakes {
		out[knownMake] = []dealer.DealerResponse{}
	}

	for _, dealer := range dealers {
		added := false
		for _, knownMake := range knownMakes {
			if !strings.Contains(dealer.SiteURL, knownMake) {
				continue
			}

			out[knownMake] = append(out[knownMake], dealer)
			added = true
		}

		if !added {
			out["unclassified"] = append(out["unclassified"], dealer)
		}
	}

	// Combine VW and Volkswagen
	tmp := mergeDealers(out["volkswagen"], out["vw"])
	delete(out, "vw")
	out["volkswagen"] = tmp

	for knownMake, dealers := range out {
		out[knownMake] = sortDealers(dealers)
	}

	return out
}

func readClassifiedDealersFile() (map[string][]dealer.DealerResponse, error) {
	out := map[string][]dealer.DealerResponse{}

	if _, err := os.Stat(classifiedDealersFile); err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
	}

	b, err := ioutil.ReadFile(classifiedDealersFile)
	if err != nil {
		return out, err
	}

	err = json.Unmarshal(b, &out)

	return out, err
}

func readClassifiedDealersFileAndFlatten() ([]dealer.DealerResponse, error) {
	out := []dealer.DealerResponse{}

	tmp, err := readClassifiedDealersFile()
	if err != nil {
		return out, err
	}

	for key := range tmp {
		out = append(out, tmp[key]...)
	}

	return sortDealers(out), nil
}

func writeClassifiedDealersFile(dealers []dealer.DealerResponse) error {
	classified := classifyDealers(dealers)

	b, err := json.Marshal(classified)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(classifiedDealersFile, b, 0755)
}

func sortDealers(dealers []dealer.DealerResponse) []dealer.DealerResponse {
	sort.Slice(dealers, func(i, j int) bool {
		return dealers[i].Name < dealers[j].Name
	})

	return dealers
}

func mergeDealers(d1, d2 []dealer.DealerResponse) []dealer.DealerResponse {
	bySiteURL := map[string]struct{}{}
	out := []dealer.DealerResponse{}

	for _, d := range d1 {
		bySiteURL[d.SiteURL] = struct{}{}
		out = append(out, d)
	}

	for _, d := range d2 {
		if _, ok := bySiteURL[d.SiteURL]; !ok {
			bySiteURL[d.SiteURL] = struct{}{}
			out = append(out, d)
		}
	}

	return sortDealers(out)
}

func loadFile() sets.String {
	b, err := ioutil.ReadFile("dealerurls.txt")
	if err != nil {
		panic(err)
	}

	urls := sets.NewString()

	for _, dealerURL := range strings.Split(string(b), "\n") {
		if dealerURL != "" {
			stripped := utils.StripHostname(dealerURL)
			urls.Insert(stripped)
		}
	}

	return urls
}

func isInterestedMake(hostname string) bool {
	for _, knownMake := range getInterestedMakes() {
		if strings.Contains(hostname, knownMake) {
			return true
		}
	}

	return false
}

func filtered(dnsName, url string) bool {
	return strings.Contains(dnsName, "dealer.com") || strings.Contains(url, dnsName) || strings.Contains(dnsName, "cloudflare.com") || !isInterestedMake(dnsName)
}

func scrapeDealerWebsite(dealerHost string, dnsHosts chan sets.String, dealerChan chan dealer.DealerResponse) {
	dnsNames, dealerByteBuf, err := doScrapeDealerWebsite(dealerHost)
	if err != nil {
		fmt.Println("Skipping:", err)
		return
	}

	go func() {
		start := time.Now()

		dealerResp, err := dealer.GetDealerResponseFromReader(dealerByteBuf, dealerHost)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Extracted contents from", utils.HostnameToURL(dealerHost), "in", time.Since(start))
		dealerChan <- dealerResp
	}()

	go func() {
		dnsHosts <- dnsNames
	}()
}

func doScrapeDealerWebsite(dealerHost string) (sets.String, *bytes.Buffer, error) {
	dnsHostsOut := sets.NewString()

	dealerURL := utils.HostnameToURL(dealerHost)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", dealerURL, nil)
	if err != nil {
		return dnsHostsOut, nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return dnsHostsOut, nil, err
	}

	defer resp.Body.Close()
	bytesBuf := bytes.NewBuffer([]byte{})

	if _, err := io.Copy(bytesBuf, resp.Body); err != nil {
		return dnsHostsOut, nil, err
	}

	return getDNSNamesFromResponse(resp, dealerURL), bytesBuf, nil
}

func getDNSNamesFromResponse(resp *http.Response, dealerURL string) sets.String {
	out := sets.NewString()

	if resp.TLS == nil {
		return out
	}

	for _, cert := range resp.TLS.PeerCertificates {
		if cert == nil {
			continue
		}

		for _, name := range cert.DNSNames {
			formatted := utils.StripHostname(name)
			if !filtered(formatted, dealerURL) {
				out.Insert(formatted)
			}
		}
	}

	return out
}

type DealerHost struct {
	Hostname string `json:"hostname"`
	Visited  bool   `json:"visited"`
}

func dealerHostsToSet(dh []DealerHost) sets.String {
	out := sets.NewString()

	for _, h := range dh {
		out.Insert(h.Hostname)
	}

	return out
}

func loadHostsFile() []DealerHost {
	dh := []DealerHost{}

	_, err := os.Stat("hosts.json")
	if !os.IsNotExist(err) {
		b, err := ioutil.ReadFile("hosts.json")
		if err != nil {
			panic(err)
		}

		if err := json.Unmarshal(b, &dh); err != nil {
			panic(err)
		}

		return sortDealerHosts(dh)
	}

	for host := range loadFile() {
		dh = append(dh, DealerHost{
			Hostname: host,
			Visited:  false,
		})
	}

	return sortDealerHosts(dh)
}

func printDealerHostStats(dh []DealerHost) {
	total := len(dh)

	visited := 0
	notVisited := 0

	for _, h := range dh {
		if h.Visited {
			visited += 1
		} else {
			notVisited += 1
		}
	}

	fmt.Println("Found", visited, "hosts we visited")
	fmt.Println("Found", notVisited, "hosts that need visited")
	fmt.Println("Total:", total)
}

func sortDealerHosts(dh []DealerHost) []DealerHost {
	sort.Slice(dh, func(i, j int) bool {
		return dh[i].Hostname < dh[j].Hostname
	})

	return dh
}

func writeHostsFile(dh []DealerHost) error {
	printDealerHostStats(dh)

	dh = sortDealerHosts(dh)

	b, err := json.Marshal(dh)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("hosts.json", b, 0755)
}

func main() {
	hostSetChan := make(chan sets.String)
	dealerHostChan := make(chan DealerHost)
	dealerRespChan := make(chan dealer.DealerResponse)

	hosts := loadHostsFile()
	printDealerHostStats(hosts)

	wp := workerpool.New(3)
	extractWP := workerpool.New(10)

	outer := func(h DealerHost) func() {
		return func() {
			hostSet, respBuf, err := doScrapeDealerWebsite(h.Hostname)
			if err != nil {
				fmt.Println("Skipping:", h.Hostname, "Error:", err)
				h.Visited = false
				dealerHostChan <- h
				return
			}

			h.Visited = true
			dealerHostChan <- h
			hostSetChan <- hostSet

			extractWP.Submit(func() {
				start := time.Now()

				dealerResp, err := dealer.GetDealerResponseFromReader(respBuf, h.Hostname)
				if err != nil {
					fmt.Println("Skipping extraction for:", h.Hostname)
				}

				fmt.Println("Content extraction for", h.Hostname, "took:", time.Since(start))

				dealerRespChan <- dealerResp
			})
		}
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		out := sets.NewString()

		known := dealerHostsToSet(hosts)

		for {
			select {
			case hostSet, ok := <-hostSetChan:
				if !ok {
					close(dealerHostChan)
					return
				}
				for foundHost := range hostSet {
					if !out.Has(foundHost) && !known.Has(foundHost) {
						if isInterestedMake(foundHost) {
							fmt.Println("Found new host:", foundHost)
							out.Insert(foundHost)
						} else {
							fmt.Println("Skipping new host:", foundHost, "Not interested make.")
						}

						dealerHostChan <- DealerHost{
							Hostname: foundHost,
							Visited:  false,
						}
					}
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		dealerResps, err := readClassifiedDealersFileAndFlatten()
		if err != nil {
			panic(err)
		}
		for {
			select {
			case dealerResp, ok := <-dealerRespChan:
				if !ok {
					if err := writeClassifiedDealersFile(dealerResps); err != nil {
						panic(err)
					}

					return
				}
				dealerResps = append(dealerResps, dealerResp)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		dealerHosts := []DealerHost{}
		for {
			select {
			case dealerHost, ok := <-dealerHostChan:
				if !ok {
					if err := writeHostsFile(dealerHosts); err != nil {
						panic(err)
					}

					return
				}

				dealerHosts = append(dealerHosts, dealerHost)
			}
		}
	}()

	/*

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case toScrape, ok := <-scrapeChan:
					if !ok {
						return
					}

					wp.Submit(outer(toScrape))
				}
			}
		}()

	*/

	for _, host := range hosts {
		if host.Visited {
			dealerHostChan <- host
			continue
		}

		if !isInterestedMake(host.Hostname) {
			fmt.Println("Skipping", host.Hostname, "because it is not an interested make")
			continue
		}

		wp.Submit(outer(host))

		//scrapeChan <- host
	}

	wp.StopWait()
	extractWP.StopWait()
	close(hostSetChan)
	close(dealerRespChan)

	wg.Wait()
}
