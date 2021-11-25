package dealer

type DealerStream struct {
	Dealer
	Err error
}

type Dealers []Dealer

type Dealer struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Address            Address  `json:"address"`
	PhoneNumber        string   `json:"phoneNumber"`
	ServicePhoneNumber string   `json:"servicePhoneNumber"`
	FaxNumber          string   `json:"faxNumber"`
	SiteURL            string   `json:"siteUrl"`
	Types              []string `json:"types"`
	Location           Location `json:"location"`
}

type Address struct {
	Type    string `json:"type"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
	City    string `json:"city"`
	County  string `json:"county"`
	State   string `json:"state"`
	Zipcode string `json:"zipcode"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Region    string  `json:"region"`
	Zone      string  `json:"zone"`
	District  string  `json:"district"`
}
