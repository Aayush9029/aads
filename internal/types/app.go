package types

type AppInfo struct {
	AdamID               int64    `json:"adamId,omitempty"`
	AppName              string   `json:"appName,omitempty"`
	CountryOrRegionCodes []string `json:"countryOrRegionCodes,omitempty"`
	DeveloperName        string   `json:"developerName,omitempty"`
}

type EligibilityRecord struct {
	AdamID          int64  `json:"adamId,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
	DeviceClass     string `json:"deviceClass,omitempty"`
	MinAge          int    `json:"minAge,omitempty"`
	State           string `json:"state,omitempty"`
	SupplySource    string `json:"supplySource,omitempty"`
}

type MediaDetail struct {
	AdamID               int64    `json:"adamId,omitempty"`
	AppName              string   `json:"appName,omitempty"`
	ArtistName           string   `json:"artistName,omitempty"`
	AvailableStorefronts []string `json:"availableStorefronts,omitempty"`
	DeviceClasses        []string `json:"deviceClasses,omitempty"`
	IconPictureURL       string   `json:"iconPictureUrl,omitempty"`
	ID                   string   `json:"id,omitempty"`
	IsPreOrder           bool     `json:"isPreOrder,omitempty"`
	PrimaryGenre         string   `json:"primaryGenre,omitempty"`
	PrimaryLanguage      string   `json:"primaryLanguage,omitempty"`
	SecondaryGenre       string   `json:"secondaryGenre,omitempty"`
}

type AppDetail = MediaDetail

type MediaLocaleDetail struct {
	AppName                    string                          `json:"appName,omitempty"`
	AppPreviewDeviceWithAssets map[string]MediaAppAssetsDetail `json:"appPreviewDeviceWithAssets,omitempty"`
	IsPrimaryLocale            bool                            `json:"isPrimaryLocale,omitempty"`
	Language                   string                          `json:"language,omitempty"`
	ShortDescription           string                          `json:"shortDescription,omitempty"`
	Subtitle                   string                          `json:"subTitle,omitempty"`
}

type LocalizedAppDetail = MediaLocaleDetail

type MediaAppAsset struct {
	AssetGenID   string `json:"assetGenId,omitempty"`
	AssetType    string `json:"assetType,omitempty"`
	AssetURL     string `json:"assetURL,omitempty"`
	Orientation  string `json:"orientation,omitempty"`
	SortPosition int64  `json:"sortPosition,omitempty"`
	SourceHeight int    `json:"sourceHeight,omitempty"`
	SourceWidth  int    `json:"sourceWidth,omitempty"`
}

type MediaAppVideoAsset struct {
	MediaAppAsset
	AssetVideoURL string `json:"assetVideoURL,omitempty"`
}

type MediaAppAssetsDetail struct {
	AppPreviewDeviceFallbackDevices []string             `json:"appPreviewDeviceFallBackDevices,omitempty"`
	AppPreviews                     []MediaAppVideoAsset `json:"appPreviews,omitempty"`
	Screenshots                     []MediaAppAsset      `json:"screenshots,omitempty"`
}
