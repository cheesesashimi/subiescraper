package dealer

type InventoryResponse struct {
	Accounts   map[string]Account   `json:"accounts"`
	Incentives map[string]Incentive `json:"incentives"`
	Inventory  []Inventory          `json:"inventory"`
	PageInfo   PageInfo             `json:"pageInfo"`
}

type InventoryAttribute struct {
	Name            string `json:"name"`
	Label           string `json:"label"`
	Value           string `json:"value"`
	LabeledValue    string `json:"labeledValue"`
	NormalizedValue string `json:"normalizedValue,omitempty"`
}

type Inventory struct {
	UUID           string   `json:"uuid"`
	AccountID      string   `json:"accountId"`
	Title          []string `json:"title"`
	OptionCodes    []string `json:"optionCodes"`
	Classification string   `json:"classification"`
	Model          string   `json:"model"`
	ModelCode      string   `json:"modelCode"`
	Link           string   `json:"link"`
	Type           int      `json:"type"`
	Status         int      `json:"status"`
	Condition      string   `json:"condition"`
	Certified      bool     `json:"certified"`
	Images         []struct {
		ID    string `json:"id"`
		URI   string `json:"uri"`
		Alt   string `json:"alt"`
		Title string `json:"title"`
	} `json:"images"`
	IncentiveIds []string             `json:"incentiveIds"`
	Attributes   []InventoryAttribute `json:"attributes"`
	Pricing      struct {
		RetailPrice string `json:"retailPrice"`
		DPrice      []struct {
			IsFinalPrice bool   `json:"isFinalPrice"`
			Label        string `json:"label"`
			Type         string `json:"type"`
			TypeClass    string `json:"typeClass"`
			Value        string `json:"value"`
		} `json:"dPrice"`
		Vehicle struct {
			Category string `json:"category"`
		} `json:"vehicle"`
		EPriceStatus string `json:"ePriceStatus"`
	} `json:"pricing"`
	Callout []struct {
		BadgeClasses []string `json:"badgeClasses"`
		ImageSrc     string   `json:"imageSrc"`
		ImageAlt     string   `json:"imageAlt"`
		ImageTitle   string   `json:"imageTitle"`
		TagName      string   `json:"tagName"`
	} `json:"callout"`
	InventoryButtons []struct {
		InventoryType           string `json:"inventoryType"`
		InventoryClassification string `json:"inventoryClassification"`
		Device                  string `json:"device"`
		BtnHref                 string `json:"btnHref"`
		BtnStyle                string `json:"btnStyle"`
		BtnTarget               string `json:"btnTarget"`
		BtnClasses              string `json:"btnClasses"`
		BtnAttributes           struct {
			DataLocation string `json:"data-location"`
			DataWidth    string `json:"data-width"`
		} `json:"btnAttributes"`
		BtnLabel        string `json:"btnLabel"`
		BtnDisabled     bool   `json:"btnDisabled"`
		ShowOnListings  bool   `json:"showOnListings"`
		ShowOnDetails   bool   `json:"showOnDetails"`
		InventoryStatus string `json:"inventoryStatus"`
	} `json:"inventoryButtons"`
	Packages []interface{} `json:"packages"`
	Videos   []interface{} `json:"videos"`
	OffSite  bool          `json:"offSite"`
	FuelType string        `json:"fuelType"`
}

type Account struct {
	Name    string `json:"name"`
	Address struct {
		City             string `json:"city"`
		Country          string `json:"country"`
		FirstLineAddress string `json:"firstLineAddress"`
		PostalCode       string `json:"postalCode"`
		State            string `json:"state"`
	} `json:"address"`
	DealerCodes []struct {
		Code     string `json:"code"`
		CodeType string `json:"codeType"`
	} `json:"dealerCodes"`
	Phone string `json:"phone"`
}

type Incentive struct {
	Conditional bool `json:"conditional"`
	Specific    struct {
		CashOption int    `json:"cashOption"`
		LenderName string `json:"lenderName"`
		Type       string `json:"_type"`
	} `json:"specific"`
	Condition         string `json:"condition"`
	Disclaimer        string `json:"disclaimer"`
	EffectiveDate     string `json:"effectiveDate"`
	ExpirationDate    string `json:"expirationDate"`
	Make              string `json:"make"`
	OfferDetails      string `json:"offerDetails"`
	ShortTitle        string `json:"shortTitle"`
	Title             string `json:"title"`
	ManufacturerOffer bool   `json:"manufacturerOffer"`
}

type TrackingData struct {
	Address struct {
		AccountName string `json:"accountName"`
		City        string `json:"city"`
		State       string `json:"state"`
		PostalCode  string `json:"postalCode"`
		Country     string `json:"country"`
	} `json:"address"`
	Certified     bool `json:"certified"`
	IndexPosition int  `json:"indexPosition"`
	ModelYear     int  `json:"modelYear"`
	Images        []struct {
		ID        int    `json:"id"`
		URI       string `json:"uri"`
		Thumbnail struct {
			URI      string        `json:"uri"`
			Provider string        `json:"provider"`
			Tags     []interface{} `json:"tags"`
		} `json:"thumbnail"`
		Provider string        `json:"provider"`
		Tags     []interface{} `json:"tags"`
	} `json:"images"`
	OptionCodes []string `json:"optionCodes"`
	DealerCodes struct {
		DealertrackPost string `json:"dealertrack-post"`
		Autocheck       string `json:"autocheck"`
		Soa             string `json:"soa"`
		Dtid            string `json:"dtid"`
		Soazone         string `json:"soazone"`
		Soadistrict     string `json:"soadistrict"`
		DtDrProfile     string `json:"dt-dr-profile"`
		SubaruExport    string `json:"subaru-export"`
		Soaregion       string `json:"soaregion"`
		AtKbb           string `json:"at-kbb"`
	} `json:"dealerCodes"`
	Pricing struct {
		Msrp       string `json:"msrp"`
		FinalPrice string `json:"finalPrice"`
	} `json:"pricing"`
	AccountID      string `json:"accountId"`
	BodyStyle      string `json:"bodyStyle"`
	ChromeID       string `json:"chromeId"`
	Classification string `json:"classification"`
	Doors          string `json:"doors"`
	DriveLine      string `json:"driveLine"`
	Engine         string `json:"engine"`
	ExteriorColor  string `json:"exteriorColor"`
	FuelType       string `json:"fuelType"`
	InteriorColor  string `json:"interiorColor"`
	InternetPrice  string `json:"internetPrice"`
	InventoryDate  string `json:"inventoryDate"`
	InventoryType  string `json:"inventoryType"`
	Link           string `json:"link"`
	Make           string `json:"make"`
	Model          string `json:"model"`
	ModelCode      string `json:"modelCode"`
	Msrp           string `json:"msrp"`
	NewOrUsed      string `json:"newOrUsed"`
	Status         string `json:"status"`
	Transmission   string `json:"transmission"`
	Trim           string `json:"trim"`
	UUID           string `json:"uuid"`
	Vin            string `json:"vin"`
	EngineSize     string `json:"engineSize,omitempty"`
}

type PageInfo struct {
	TotalCount          int            `json:"totalCount"`
	PageSize            int            `json:"pageSize"`
	PageStart           int            `json:"pageStart"`
	EnableMyCars        bool           `json:"enableMyCars"`
	EnableMyCarsOnVLP   bool           `json:"enableMyCarsOnVLP"`
	EnableMediaCarousel bool           `json:"enableMediaCarousel"`
	TrackingData        []TrackingData `json:"trackingData"`
}
