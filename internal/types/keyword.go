package types

type Keyword struct {
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	BidAmount        *Money `json:"bidAmount,omitempty"`
	CampaignID       int64  `json:"campaignId,omitempty"`
	CreationTime     string `json:"creationTime,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
	ID               int64  `json:"id,omitempty"`
	MatchType        string `json:"matchType"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Status           string `json:"status,omitempty"`
	Text             string `json:"text"`
}

type KeywordUpdateRequest struct {
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	BidAmount        *Money `json:"bidAmount,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
	ID               int64  `json:"id,omitempty"`
	MatchType        string `json:"matchType"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Status           string `json:"status,omitempty"`
	Text             string `json:"text"`
}

type NegativeKeyword struct {
	AdGroupID        int64  `json:"adGroupId,omitempty"`
	CampaignID       int64  `json:"campaignId,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
	ID               int64  `json:"id,omitempty"`
	MatchType        string `json:"matchType"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Status           string `json:"status,omitempty"`
	Text             string `json:"text"`
}
