package types

type SearchEntity struct {
	AdminArea       string `json:"adminArea,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	Entity          string `json:"entity,omitempty"`
	ID              string `json:"id,omitempty"`
	Locality        string `json:"locality,omitempty"`
}

type GeoRequest struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
}
