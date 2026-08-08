package types

type Ad struct {
	AdGroupID           int64    `json:"adGroupId,omitempty"`
	CampaignID          int64    `json:"campaignId,omitempty"`
	CreationTime        string   `json:"creationTime,omitempty"`
	CreativeID          int64    `json:"creativeId,omitempty"`
	CreativeType        string   `json:"creativeType,omitempty"`
	Deleted             bool     `json:"deleted,omitempty"`
	ID                  int64    `json:"id,omitempty"`
	ModificationTime    string   `json:"modificationTime,omitempty"`
	Name                string   `json:"name,omitempty"`
	OrgID               int64    `json:"orgId,omitempty"`
	ServingStateReasons []string `json:"servingStateReasons,omitempty"`
	ServingStatus       string   `json:"servingStatus,omitempty"`
	Status              string   `json:"status,omitempty"`
}

type AdCreate struct {
	CreativeID int64  `json:"creativeId"`
	Name       string `json:"name"`
	Status     string `json:"status"`
}

type AdUpdate struct {
	CreativeID int64  `json:"creativeId,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status,omitempty"`
}
