package types

import "encoding/json"

type ReportingRequest struct {
	EndTime                    string    `json:"endTime"`
	Granularity                string    `json:"granularity,omitempty"`
	GroupBy                    []string  `json:"groupBy,omitempty"`
	ReturnGrandTotals          bool      `json:"returnGrandTotals,omitempty"`
	ReturnRecordsWithNoMetrics bool      `json:"returnRecordsWithNoMetrics,omitempty"`
	ReturnRowTotals            bool      `json:"returnRowTotals,omitempty"`
	Selector                   *Selector `json:"selector"`
	StartTime                  string    `json:"startTime"`
	TimeZone                   string    `json:"timeZone,omitempty"`
}

type ReportingResponse struct {
	ReportingDataResponse *ReportingDataResponse `json:"reportingDataResponse,omitempty"`
}

type ReportingDataResponse struct {
	GrandTotals *GrandTotalsRow `json:"grandTotals,omitempty"`
	Row         []ReportRow     `json:"row,omitempty"`
}

type GrandTotalsRow struct {
	Other bool      `json:"other,omitempty"`
	Total *SpendRow `json:"total,omitempty"`
}

type ReportRow struct {
	Granularity []ExtendedSpendRow         `json:"granularity,omitempty"`
	Insights    map[string]json.RawMessage `json:"insights,omitempty"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
	Other       bool                       `json:"other,omitempty"`
	Total       *SpendRow                  `json:"total,omitempty"`
}

type Row = ReportRow

type SpendRow struct {
	AvgCPM               *Money  `json:"avgCPM,omitempty"`
	AvgCPT               *Money  `json:"avgCPT,omitempty"`
	Impressions          int64   `json:"impressions,omitempty"`
	LocalSpend           *Money  `json:"localSpend,omitempty"`
	TapInstallCPI        *Money  `json:"tapInstallCPI,omitempty"`
	TapInstallRate       float64 `json:"tapInstallRate,omitempty"`
	TapInstalls          int64   `json:"tapInstalls,omitempty"`
	TapNewDownloads      int64   `json:"tapNewDownloads,omitempty"`
	TapPreOrdersPlaced   int64   `json:"tapPreOrdersPlaced,omitempty"`
	TapRedownloads       int64   `json:"tapRedownloads,omitempty"`
	Taps                 int64   `json:"taps,omitempty"`
	TotalAvgCPI          *Money  `json:"totalAvgCPI,omitempty"`
	TotalInstallRate     float64 `json:"totalInstallRate,omitempty"`
	TotalInstalls        int64   `json:"totalInstalls,omitempty"`
	TotalNewDownloads    int64   `json:"totalNewDownloads,omitempty"`
	TotalPreOrdersPlaced int64   `json:"totalPreOrdersPlaced,omitempty"`
	TotalRedownloads     int64   `json:"totalRedownloads,omitempty"`
	TTR                  float64 `json:"ttr,omitempty"`
	ViewInstalls         int64   `json:"viewInstalls,omitempty"`
	ViewNewDownloads     int64   `json:"viewNewDownloads,omitempty"`
	ViewPreOrdersPlaced  int64   `json:"viewPreOrdersPlaced,omitempty"`
	ViewRedownloads      int64   `json:"viewRedownloads,omitempty"`
}

type ExtendedSpendRow struct {
	SpendRow
	Date string `json:"date,omitempty"`
}

type GranularityRow = ExtendedSpendRow

type ReportingCampaign struct {
	AdChannelType                      string              `json:"adChannelType,omitempty"`
	App                                *CampaignAppDetail  `json:"app,omitempty"`
	BiddingStrategy                    string              `json:"biddingStrategy,omitempty"`
	CampaignID                         int64               `json:"campaignId,omitempty"`
	CampaignName                       string              `json:"campaignName,omitempty"`
	CampaignStatus                     string              `json:"campaignStatus,omitempty"`
	CountriesOrRegions                 []string            `json:"countriesOrRegions,omitempty"`
	CountryOrRegionServingStateReasons map[string][]string `json:"countryOrRegionServingStateReasons,omitempty"`
	DailyBudget                        *Money              `json:"dailyBudget,omitempty"`
	Deleted                            bool                `json:"deleted,omitempty"`
	DisplayStatus                      string              `json:"displayStatus,omitempty"`
	ModificationTime                   string              `json:"modificationTime,omitempty"`
	OrgID                              int64               `json:"orgId,omitempty"`
	ServingStateReasons                []string            `json:"servingStateReasons,omitempty"`
	ServingStatus                      string              `json:"servingStatus,omitempty"`
	SupplySources                      []string            `json:"supplySources,omitempty"`
	TargetCPA                          *Money              `json:"targetCpa,omitempty"`
	TotalBudget                        *Money              `json:"totalBudget,omitempty"`
}

type ReportingAdGroup struct {
	AdGroupDisplayStatus       string   `json:"adGroupDisplayStatus,omitempty"`
	AdGroupID                  int64    `json:"adGroupId,omitempty"`
	AdGroupName                string   `json:"adGroupName,omitempty"`
	AdGroupServingStateReasons []string `json:"adGroupServingStateReasons,omitempty"`
	AdGroupServingStatus       string   `json:"adGroupServingStatus,omitempty"`
	AdGroupStatus              string   `json:"adGroupStatus,omitempty"`
	AutomatedKeywordsOptIn     bool     `json:"automatedKeywordsOptIn,omitempty"`
	AutomatedKeywordsRequired  bool     `json:"automatedKeywordsRequired,omitempty"`
	BiddingStrategy            string   `json:"biddingStrategy,omitempty"`
	CampaignID                 int64    `json:"campaignId,omitempty"`
	CPAGoal                    *Money   `json:"cpaGoal,omitempty"`
	DefaultBidAmount           *Money   `json:"defaultBidAmount,omitempty"`
	Deleted                    bool     `json:"deleted,omitempty"`
	EndTime                    string   `json:"endTime,omitempty"`
	ModificationTime           string   `json:"modificationTime,omitempty"`
	OrgID                      int64    `json:"orgId,omitempty"`
	StartTime                  string   `json:"startTime,omitempty"`
}

type ReportingKeyword struct {
	AdGroupDeleted       bool   `json:"adGroupDeleted,omitempty"`
	AdGroupID            int64  `json:"adGroupId,omitempty"`
	AdGroupName          string `json:"adGroupName,omitempty"`
	BidAmount            *Money `json:"bidAmount,omitempty"`
	CampaignID           int64  `json:"campaignId,omitempty"`
	Deleted              bool   `json:"deleted,omitempty"`
	Keyword              string `json:"keyword,omitempty"`
	KeywordDisplayStatus string `json:"keywordDisplayStatus,omitempty"`
	KeywordID            int64  `json:"keywordId,omitempty"`
	KeywordStatus        string `json:"keywordStatus,omitempty"`
	MatchType            string `json:"matchType,omitempty"`
	ModificationTime     string `json:"modificationTime,omitempty"`
	OrgID                int64  `json:"orgId,omitempty"`
}

type ReportingSearchTerm struct {
	ReportingKeyword
	SearchTermSource string `json:"searchTermSource,omitempty"`
	SearchTermText   string `json:"searchTermText,omitempty"`
}

type ReportingAd struct {
	AdDisplayStatus       string `json:"adDisplayStatus,omitempty"`
	AdGroupID             int64  `json:"adGroupId,omitempty"`
	AdID                  int64  `json:"adId,omitempty"`
	AdName                string `json:"adName,omitempty"`
	AdServingStateReasons string `json:"adServingStateReasons,omitempty"`
	CampaignID            int64  `json:"campaignId,omitempty"`
	CreationTime          string `json:"creationTime,omitempty"`
	CreativeID            int64  `json:"creativeId,omitempty"`
	CreativeType          string `json:"creativeType,omitempty"`
	Deleted               bool   `json:"deleted,omitempty"`
	Language              string `json:"language,omitempty"`
	ModificationTime      string `json:"modificationTime,omitempty"`
	OrgID                 int64  `json:"orgId,omitempty"`
	ProductPageID         string `json:"productPageId,omitempty"`
	Status                string `json:"status,omitempty"`
}

type CampaignAppDetail struct {
	AdamID  int64  `json:"adamId,omitempty"`
	AppName string `json:"appName,omitempty"`
}

type Insights struct {
	BidRecommendation *BidRecommendation         `json:"bidRecommendation,omitempty"`
	AdditionalFields  map[string]json.RawMessage `json:"-"`
}

type BidRecommendation struct {
	SuggestedBidAmount *Money `json:"suggestedBidAmount,omitempty"`
}

type KeywordInsights = Insights
type KeywordBidRecommendation = BidRecommendation
