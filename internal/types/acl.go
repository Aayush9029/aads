package types

type UserACL struct {
	Currency     string   `json:"currency,omitempty"`
	OrgID        int64    `json:"orgId,omitempty"`
	OrgName      string   `json:"orgName,omitempty"`
	ParentOrgID  int64    `json:"parentOrgId,omitempty"`
	PaymentModel string   `json:"paymentModel,omitempty"`
	RoleNames    []string `json:"roleNames,omitempty"`
	TimeZone     string   `json:"timeZone,omitempty"`
}

type MeDetail struct {
	ParentOrgID int64 `json:"parentOrgId,omitempty"`
	UserID      int64 `json:"userId,omitempty"`
}
