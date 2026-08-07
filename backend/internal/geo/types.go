package geo

type ResourceStatus struct {
	Name         string `json:"name"`
	Available    bool   `json:"available"`
	Size         int64  `json:"size"`
	ModifiedAt   string `json:"modifiedAt,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Version      string `json:"version,omitempty"`
	DatabaseType string `json:"databaseType,omitempty"`
	BuildEpoch   uint   `json:"buildEpoch,omitempty"`
	Error        string `json:"error,omitempty"`
}

type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type CategoryInfo struct {
	File            string `json:"file"`
	SupportsReverse bool   `json:"supportsReverse"`
}

type CategoriesResponse struct {
	GeoSite []Category   `json:"geosite"`
	GeoIP   []Category   `json:"geoip"`
	MetaDB  CategoryInfo `json:"metadb"`
	ASN     CategoryInfo `json:"asn"`
}

type GeoSiteHit struct {
	Category string `json:"category"`
	Type     string `json:"type"`
	Value    string `json:"value"`
}

type IPHit struct {
	IP       string `json:"ip"`
	Category string `json:"category"`
	CIDR     string `json:"cidr"`
}

type DBHit struct {
	IP     string `json:"ip"`
	CIDR   string `json:"cidr"`
	Record string `json:"record"`
}

type ASNHit struct {
	IP           string `json:"ip"`
	CIDR         string `json:"cidr"`
	ASN          uint32 `json:"asn"`
	Organization string `json:"organization"`
}

type IPGeoFlag struct {
	Img          string `json:"img,omitempty"`
	Emoji        string `json:"emoji,omitempty"`
	EmojiUnicode string `json:"emoji_unicode,omitempty"`
}

type IPGeoResponse struct {
	IP            string    `json:"ip"`
	Continent     string    `json:"continent,omitempty"`
	ContinentCode string    `json:"continent_code,omitempty"`
	Country       string    `json:"country,omitempty"`
	CountryCode   string    `json:"countryCode,omitempty"`
	Region        string    `json:"region,omitempty"`
	RegionCode    string    `json:"region_code,omitempty"`
	City          string    `json:"city,omitempty"`
	Postal        string    `json:"postal,omitempty"`
	Flag          IPGeoFlag `json:"flag,omitempty"`
	Latitude      float64   `json:"latitude,omitempty"`
	Longitude     float64   `json:"longitude,omitempty"`
	ASN           string    `json:"asn,omitempty"`
	Organization  string    `json:"organization,omitempty"`
	ISP           string    `json:"isp,omitempty"`
}

type QueryResponse struct {
	Domain         string       `json:"domain"`
	InputType      string       `json:"inputType,omitempty"`
	IPs            []string     `json:"ips"`
	ResolveSkipped bool         `json:"resolveSkipped"`
	ResolveError   string       `json:"resolveError,omitempty"`
	GeoSite        []GeoSiteHit `json:"geosite"`
	GeoIP          []IPHit      `json:"geoip"`
	MetaDB         []DBHit      `json:"metadb"`
	ASN            []ASNHit     `json:"asn"`
}

type ReverseItem struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type ReverseResponse struct {
	File     string        `json:"file"`
	Category string        `json:"category"`
	Items    []ReverseItem `json:"items,omitempty"`
	Total    int           `json:"total"`
	Limit    int           `json:"limit"`
	Offset   int           `json:"offset"`
	Message  string        `json:"message,omitempty"`
}

type SearchRequest struct {
	File    string `json:"file"`
	Field   string `json:"field"`
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

type SearchItem struct {
	Type   string `json:"type"`
	Value  string `json:"value"`
	Detail string `json:"detail,omitempty"`
}

type SearchResponse struct {
	File    string       `json:"file"`
	Field   string       `json:"field"`
	Keyword string       `json:"keyword"`
	Items   []SearchItem `json:"items,omitempty"`
	Total   int          `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	Message string       `json:"message,omitempty"`
}

type UpdateItem struct {
	Name    string `json:"name"`
	Updated bool   `json:"updated"`
	Error   string `json:"error,omitempty"`
}

type UpdateResponse struct {
	Items []UpdateItem `json:"items"`
}
