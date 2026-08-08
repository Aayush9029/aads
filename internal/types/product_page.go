package types

type ProductPageDetail struct {
	AdamID           int64  `json:"adamId,omitempty"`
	CreationTime     string `json:"creationTime,omitempty"`
	DeepLink         string `json:"deepLink,omitempty"`
	ID               string `json:"id,omitempty"`
	ModificationTime string `json:"modificationTime,omitempty"`
	Name             string `json:"name,omitempty"`
	State            string `json:"state,omitempty"`
}

type ProductPageDetailWithAssets struct {
	AdamID            int64                            `json:"adamId,omitempty"`
	ContentProviderID int64                            `json:"contentProviderId,omitempty"`
	CreationTime      string                           `json:"creationTime,omitempty"`
	ID                int64                            `json:"id,omitempty"`
	IsDefault         bool                             `json:"isDefault,omitempty"`
	Localization      []CreativeLocalizationWithAssets `json:"localization,omitempty"`
	ModificationTime  string                           `json:"modificationTime,omitempty"`
	Name              string                           `json:"name,omitempty"`
}

type ProductPageLocaleDetail struct {
	AdamID                     int64                           `json:"adamId,omitempty"`
	AppName                    string                          `json:"appName,omitempty"`
	AppPreviewDeviceWithAssets map[string]MediaAppAssetsDetail `json:"appPreviewDeviceWithAssets,omitempty"`
	DeviceClasses              string                          `json:"deviceClasses,omitempty"`
	Language                   string                          `json:"language,omitempty"`
	LanguageCode               string                          `json:"languageCode,omitempty"`
	ProductPageID              string                          `json:"productPageId,omitempty"`
	PromotionalText            string                          `json:"promotionalText,omitempty"`
	ShortDescription           string                          `json:"shortDescription,omitempty"`
	Subtitle                   string                          `json:"subTitle,omitempty"`
}

type LocaleInfo struct {
	Language     string `json:"language,omitempty"`
	LanguageCode string `json:"languageCode,omitempty"`
}

type CountryOrRegion struct {
	CountryOrRegion    string       `json:"countryOrRegion,omitempty"`
	DefaultLanguages   []LocaleInfo `json:"defaultLanguages,omitempty"`
	SupportedLanguages []LocaleInfo `json:"supportedLanguages,omitempty"`
}

type DeviceSizeMapping map[string]string

type AppPreviewDevicesMappingResponse struct {
	Data DeviceSizeMapping `json:"data,omitempty"`
}
