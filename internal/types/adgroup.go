package types

type AdGroup struct {
	AutomatedKeywordsOptIn    bool                 `json:"automatedKeywordsOptIn,omitempty"`
	AutomatedKeywordsRequired bool                 `json:"automatedKeywordsRequired,omitempty"`
	BiddingStrategy           string               `json:"biddingStrategy,omitempty"`
	CampaignID                int64                `json:"campaignId,omitempty"`
	CPAGoal                   *Money               `json:"cpaGoal,omitempty"`
	CreationTime              string               `json:"creationTime,omitempty"`
	DefaultBidAmount          *Money               `json:"defaultBidAmount,omitempty"`
	Deleted                   bool                 `json:"deleted,omitempty"`
	DisplayStatus             string               `json:"displayStatus,omitempty"`
	EndTime                   string               `json:"endTime,omitempty"`
	ID                        int64                `json:"id,omitempty"`
	ModificationTime          string               `json:"modificationTime,omitempty"`
	Name                      string               `json:"name,omitempty"`
	OrgID                     int64                `json:"orgId,omitempty"`
	PaymentModel              string               `json:"paymentModel,omitempty"`
	PricingModel              string               `json:"pricingModel,omitempty"`
	ServingStateReasons       []string             `json:"servingStateReasons,omitempty"`
	ServingStatus             string               `json:"servingStatus,omitempty"`
	StartTime                 string               `json:"startTime,omitempty"`
	Status                    string               `json:"status,omitempty"`
	TargetingDimensions       *TargetingDimensions `json:"targetingDimensions,omitempty"`
}

type AdGroupCreate struct {
	AutomatedKeywordsOptIn    bool                 `json:"automatedKeywordsOptIn,omitempty"`
	AutomatedKeywordsRequired bool                 `json:"automatedKeywordsRequired,omitempty"`
	CampaignID                int64                `json:"campaignId,omitempty"`
	CPAGoal                   *Money               `json:"cpaGoal,omitempty"`
	DefaultBidAmount          *Money               `json:"defaultBidAmount,omitempty"`
	EndTime                   string               `json:"endTime,omitempty"`
	Name                      string               `json:"name"`
	OrgID                     int64                `json:"orgId,omitempty"`
	PricingModel              string               `json:"pricingModel,omitempty"`
	StartTime                 string               `json:"startTime,omitempty"`
	Status                    string               `json:"status,omitempty"`
	TargetingDimensions       *TargetingDimensions `json:"targetingDimensions,omitempty"`
}

type AdGroupUpdate struct {
	AutomatedKeywordsOptIn *bool                `json:"automatedKeywordsOptIn,omitempty"`
	CPAGoal                *Money               `json:"cpaGoal,omitempty"`
	DefaultBidAmount       *Money               `json:"defaultBidAmount,omitempty"`
	EndTime                string               `json:"endTime,omitempty"`
	Name                   string               `json:"name,omitempty"`
	StartTime              string               `json:"startTime,omitempty"`
	Status                 string               `json:"status,omitempty"`
	TargetingDimensions    *TargetingDimensions `json:"targetingDimensions,omitempty"`
}

type TargetingDimensions struct {
	AdminArea      *LocationCriteria       `json:"adminArea,omitempty"`
	Age            *AgeCriteria            `json:"age,omitempty"`
	AppCategories  *AppCategoriesCriteria  `json:"appCategories,omitempty"`
	AppDownloaders *AppDownloadersCriteria `json:"appDownloaders,omitempty"`
	Country        *LocationCriteria       `json:"country,omitempty"`
	Daypart        *DaypartCriteria        `json:"daypart,omitempty"`
	DeviceClass    *DeviceClassCriteria    `json:"deviceClass,omitempty"`
	Gender         *GenderCriteria         `json:"gender,omitempty"`
	Locality       *LocationCriteria       `json:"locality,omitempty"`
}

type AgeCriteria struct {
	Included []AgeRange `json:"included,omitempty"`
}

type AgeRange struct {
	MaxAge int `json:"maxAge,omitempty"`
	MinAge int `json:"minAge,omitempty"`
}

type GenderCriteria struct {
	Included []string `json:"included,omitempty"`
}

type DeviceClassCriteria struct {
	Included []string `json:"included,omitempty"`
}

type DaypartCriteria struct {
	UserTime *DaypartDetail `json:"userTime,omitempty"`
}

type DaypartDetail struct {
	Included []int `json:"included,omitempty"`
}

type LocationCriteria struct {
	Included []string `json:"included,omitempty"`
}

type AdminAreaCriteria = LocationCriteria
type CountryCriteria = LocationCriteria
type LocalityCriteria = LocationCriteria

type AppDownloadersCriteria struct {
	Excluded []string `json:"excluded,omitempty"`
	Included []string `json:"included,omitempty"`
}

type AppCategoriesCriteria struct {
	Excluded []int64 `json:"excluded,omitempty"`
	Included []int64 `json:"included,omitempty"`
}

type AppCategoryCriteria = AppCategoriesCriteria
type AppDownloaderCriteria = AppDownloadersCriteria
