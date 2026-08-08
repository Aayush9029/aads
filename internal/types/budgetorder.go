package types

type BudgetOrder struct {
	BillingEmail      string   `json:"billingEmail,omitempty"`
	Budget            *Money   `json:"budget,omitempty"`
	ClientName        string   `json:"clientName,omitempty"`
	EndDate           string   `json:"endDate,omitempty"`
	ID                int64    `json:"id,omitempty"`
	Name              string   `json:"name,omitempty"`
	OrderNumber       string   `json:"orderNumber,omitempty"`
	ParentOrgID       int64    `json:"parentOrgId,omitempty"`
	PrimaryBuyerEmail string   `json:"primaryBuyerEmail,omitempty"`
	PrimaryBuyerName  string   `json:"primaryBuyerName,omitempty"`
	StartDate         string   `json:"startDate,omitempty"`
	Status            string   `json:"status,omitempty"`
	SupplySources     []string `json:"supplySources,omitempty"`
}

type BudgetOrderInfo struct {
	BO     *BudgetOrder `json:"bo,omitempty"`
	OrgIDs []int64      `json:"orgIds,omitempty"`
}

type BudgetOrderCreate struct {
	BO     *BudgetOrderCreateDetail `json:"bo"`
	OrgIDs []int64                  `json:"orgIds"`
}

type BudgetOrderCreateDetail struct {
	BillingEmail      string   `json:"billingEmail,omitempty"`
	Budget            *Money   `json:"budget,omitempty"`
	ClientName        string   `json:"clientName,omitempty"`
	EndDate           string   `json:"endDate,omitempty"`
	Name              string   `json:"name,omitempty"`
	OrderNumber       string   `json:"orderNumber,omitempty"`
	PrimaryBuyerEmail string   `json:"primaryBuyerEmail,omitempty"`
	PrimaryBuyerName  string   `json:"primaryBuyerName,omitempty"`
	StartDate         string   `json:"startDate,omitempty"`
	SupplySources     []string `json:"supplySources,omitempty"`
}

type BudgetOrderUpdate struct {
	BO     *BudgetOrderUpdateDetail `json:"bo"`
	OrgIDs []int64                  `json:"orgIds,omitempty"`
}

type BudgetOrderUpdateDetail struct {
	BillingEmail      string `json:"billingEmail,omitempty"`
	Budget            *Money `json:"budget,omitempty"`
	ClientName        string `json:"clientName,omitempty"`
	EndDate           string `json:"endDate,omitempty"`
	Name              string `json:"name,omitempty"`
	OrderNumber       string `json:"orderNumber,omitempty"`
	PrimaryBuyerEmail string `json:"primaryBuyerEmail,omitempty"`
	PrimaryBuyerName  string `json:"primaryBuyerName,omitempty"`
	StartDate         string `json:"startDate,omitempty"`
}
