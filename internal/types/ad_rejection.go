package types

type ProductPageReason struct {
	AdamID          int64  `json:"adamId,omitempty"`
	AssetGenID      string `json:"assetGenId,omitempty"`
	Comment         string `json:"comment,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
	ID              int64  `json:"id,omitempty"`
	LanguageCode    string `json:"languageCode,omitempty"`
	ProductPageID   string `json:"productPageId,omitempty"`
	ReasonCode      string `json:"reasonCode,omitempty"`
	ReasonLevel     string `json:"reasonLevel,omitempty"`
	ReasonType      string `json:"reasonType,omitempty"`
	SupplySource    string `json:"supplySource,omitempty"`
}

type ProductPageReasonCreate struct {
	AdamID          int64  `json:"adamId"`
	AssetGenID      string `json:"assetGenId,omitempty"`
	Comment         string `json:"comment,omitempty"`
	CountryOrRegion string `json:"countryOrRegion"`
	ID              int64  `json:"id,omitempty"`
	LanguageCode    string `json:"languageCode"`
	ProductPageID   string `json:"productPageId"`
	ReasonCode      string `json:"reasonCode"`
	ReasonLevel     string `json:"reasonLevel,omitempty"`
	ReasonType      string `json:"reasonType"`
	SupplySource    string `json:"supplySource"`
}

type AppAsset struct {
	AdamID           string `json:"adamId,omitempty"`
	AppPreviewDevice string `json:"appPreviewDevice,omitempty"`
	AssetGenID       string `json:"assetGenId,omitempty"`
	AssetType        string `json:"assetType,omitempty"`
	AssetURL         string `json:"assetURL,omitempty"`
	AssetVideoURL    string `json:"assetVideoUrl,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
	Orientation      string `json:"orientation,omitempty"`
	SourceHeight     int    `json:"sourceHeight,omitempty"`
	SourceWidth      int    `json:"sourceWidth,omitempty"`
}
