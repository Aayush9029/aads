package types

type CustomReportRequest struct {
	DateRange   string       `json:"dateRange,omitempty"`
	EndTime     string       `json:"endTime,omitempty"`
	Granularity string       `json:"granularity,omitempty"`
	Name        string       `json:"name"`
	Selector    *SovSelector `json:"selector,omitempty"`
	StartTime   string       `json:"startTime,omitempty"`
}

type SovSelector struct {
	Conditions []*SovCondition `json:"conditions,omitempty"`
}

type SovCondition struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type CustomReportResponse struct {
	CreationTime     string       `json:"creationTime,omitempty"`
	Dimensions       []string     `json:"dimensions,omitempty"`
	DownloadURI      string       `json:"downloadUri,omitempty"`
	EndTime          string       `json:"endTime,omitempty"`
	Granularity      string       `json:"granularity,omitempty"`
	ID               int64        `json:"id,omitempty"`
	Metrics          []string     `json:"metrics,omitempty"`
	ModificationTime string       `json:"modificationTime,omitempty"`
	Name             string       `json:"name,omitempty"`
	Selector         *SovSelector `json:"selector,omitempty"`
	StartTime        string       `json:"startTime,omitempty"`
	State            string       `json:"state,omitempty"`
}
