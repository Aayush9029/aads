package types

type Campaign struct {
	AdamID                             int64               `json:"adamId"`
	AdChannelType                      string              `json:"adChannelType"`
	BillingEvent                       string              `json:"billingEvent"`
	BiddingStrategy                    string              `json:"biddingStrategy,omitempty"`
	BudgetOrders                       []int64             `json:"budgetOrders,omitempty"`
	CountriesOrRegions                 []string            `json:"countriesOrRegions"`
	CountryOrRegionServingStateReasons map[string][]string `json:"countryOrRegionServingStateReasons,omitempty"`
	CreationTime                       string              `json:"creationTime,omitempty"`
	DailyBudgetAmount                  *Money              `json:"dailyBudgetAmount"`
	Deleted                            bool                `json:"deleted,omitempty"`
	DisplayStatus                      string              `json:"displayStatus,omitempty"`
	EndTime                            string              `json:"endTime,omitempty"`
	ID                                 int64               `json:"id,omitempty"`
	LOCInvoiceDetails                  *LOCInvoiceDetails  `json:"locInvoiceDetails,omitempty"`
	ModificationTime                   string              `json:"modificationTime,omitempty"`
	Name                               string              `json:"name"`
	OrgID                              int64               `json:"orgId,omitempty"`
	PaymentModel                       string              `json:"paymentModel,omitempty"`
	SapinLawResponse                   string              `json:"sapinLawResponse,omitempty"`
	ServingStateReasons                []string            `json:"servingStateReasons,omitempty"`
	ServingStatus                      string              `json:"servingStatus,omitempty"`
	StartTime                          string              `json:"startTime,omitempty"`
	Status                             string              `json:"status,omitempty"`
	SupplySources                      []string            `json:"supplySources"`
	TargetCPA                          *Money              `json:"targetCpa,omitempty"`
}

type CampaignCreate struct {
	AdamID             int64              `json:"adamId"`
	AdChannelType      string             `json:"adChannelType"`
	BillingEvent       string             `json:"billingEvent"`
	BiddingStrategy    string             `json:"biddingStrategy,omitempty"`
	BudgetOrders       []int64            `json:"budgetOrders,omitempty"`
	CountriesOrRegions []string           `json:"countriesOrRegions"`
	DailyBudgetAmount  *Money             `json:"dailyBudgetAmount"`
	EndTime            string             `json:"endTime,omitempty"`
	LOCInvoiceDetails  *LOCInvoiceDetails `json:"locInvoiceDetails,omitempty"`
	Name               string             `json:"name"`
	OrgID              int64              `json:"orgId,omitempty"`
	StartTime          string             `json:"startTime,omitempty"`
	Status             string             `json:"status,omitempty"`
	SupplySources      []string           `json:"supplySources"`
	TargetCPA          *Money             `json:"targetCpa,omitempty"`
}

type CampaignUpdate struct {
	BiddingStrategy    string             `json:"biddingStrategy,omitempty"`
	BudgetOrders       []int64            `json:"budgetOrders,omitempty"`
	CountriesOrRegions []string           `json:"countriesOrRegions,omitempty"`
	DailyBudgetAmount  *Money             `json:"dailyBudgetAmount,omitempty"`
	LOCInvoiceDetails  *LOCInvoiceDetails `json:"locInvoiceDetails,omitempty"`
	Name               string             `json:"name,omitempty"`
	Status             string             `json:"status,omitempty"`
	TargetCPA          *Money             `json:"targetCpa,omitempty"`
}

type UpdateCampaignRequest struct {
	Campaign                                 *CampaignUpdate `json:"campaign"`
	ClearGeoTargetingOnCountryOrRegionChange bool            `json:"clearGeoTargetingOnCountryOrRegionChange,omitempty"`
}
