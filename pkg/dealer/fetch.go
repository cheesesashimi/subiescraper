package dealer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const dealersByStateEndpoint string = "https://www.subaru.com/services/dealers/by/state"

func ByState(state string) (Dealers, error) {
	out := Dealers{}

	u, err := url.Parse(dealersByStateEndpoint)
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
		return out, fmt.Errorf("could not parse dealer respones: %w", err)
	}

	return out, nil
}

func ByStates(states []string) chan DealerStream {
	dealerStream := make(chan DealerStream)

	go func() {
		for _, state := range states {
			dealers, err := ByState(state)
			if err != nil {
				dealerStream <- DealerStream{
					Err: err,
				}
				break
			}

			for _, d := range dealers {
				dealerStream <- DealerStream{
					Dealer: d,
					Err:    nil,
				}
			}
		}

		close(dealerStream)
	}()

	return dealerStream
}
