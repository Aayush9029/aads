package types

type Creative struct {
	AdamID           int64    `json:"adamId,omitempty"`
	CreationTime     string   `json:"creationTime,omitempty"`
	ID               int64    `json:"id,omitempty"`
	ModificationTime string   `json:"modificationTime,omitempty"`
	Name             string   `json:"name,omitempty"`
	OrgID            int64    `json:"orgId,omitempty"`
	ProductPageID    string   `json:"productPageId,omitempty"`
	State            string   `json:"state,omitempty"`
	StateReasons     []string `json:"stateReasons,omitempty"`
	Type             string   `json:"type,omitempty"`
}

type CustomProductPageCreative struct {
	AdamID           int64    `json:"adamId"`
	CreationTime     string   `json:"creationTime,omitempty"`
	ID               int64    `json:"id,omitempty"`
	ModificationTime string   `json:"modificationTime,omitempty"`
	Name             string   `json:"name"`
	OrgID            int64    `json:"orgId,omitempty"`
	ProductPageID    string   `json:"productPageId"`
	State            string   `json:"state,omitempty"`
	StateReasons     []string `json:"stateReasons,omitempty"`
	Type             string   `json:"type"`
}

type DefaultProductPageCreative struct {
	AdamID           int64    `json:"adamId"`
	CreationTime     string   `json:"creationTime,omitempty"`
	ID               int64    `json:"id,omitempty"`
	ModificationTime string   `json:"modificationTime,omitempty"`
	Name             string   `json:"name"`
	OrgID            int64    `json:"orgId,omitempty"`
	ProductPageID    string   `json:"productPageId,omitempty"`
	State            string   `json:"state,omitempty"`
	StateReasons     []string `json:"stateReasons,omitempty"`
	Type             string   `json:"type"`
}

type CreativeCreate struct {
	AdamID        int64  `json:"adamId"`
	Name          string `json:"name"`
	ProductPageID string `json:"productPageId,omitempty"`
	Type          string `json:"type"`
}

type CreativeLocalization struct {
	AppName          string `json:"appName,omitempty"`
	DeviceClasses    string `json:"deviceClasses,omitempty"`
	Language         string `json:"language,omitempty"`
	LanguageCode     string `json:"languageCode,omitempty"`
	PromotionalText  string `json:"promotionalText,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	Subtitle         string `json:"subTitle,omitempty"`
}

type CreativeLocalizationWithAssets struct {
	CreativeLocalization
	AppPreviewDeviceWithAssets map[string]MediaAppAssetsDetail `json:"appPreviewDeviceWithAssets,omitempty"`
}
