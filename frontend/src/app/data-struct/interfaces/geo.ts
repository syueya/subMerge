export interface GeoIPGeoFlag {
  img?: string;
  emoji?: string;
  emoji_unicode?: string;
}

export interface GeoIPGeoResponse {
  ip: string;
  flag?: GeoIPGeoFlag;
  continent?: string;
  continent_code?: string;
  country?: string;
  countryCode?: string;
  region?: string;
  region_code?: string;
  city?: string;
  postal?: string;
  latitude?: number;
  longitude?: number;
  asn?: string;
  organization?: string;
  isp?: string;
}
export interface GeoStatus {
  name: string;
  available: boolean;
  size: number;
  modifiedAt?: string;
  sha256?: string;
  version?: string;
  databaseType?: string;
  buildEpoch?: number;
  error?: string;
}

export interface GeoCategory {
  name: string;
  count: number;
}

export interface GeoCategoryInfo {
  file: string;
  supportsReverse: boolean;
}

export interface GeoCategoriesResponse {
  geosite: GeoCategory[];
  geoip: GeoCategory[];
  metadb: GeoCategoryInfo;
  asn: GeoCategoryInfo;
}

export interface GeoSiteHit {
  category: string;
  type: string;
  value: string;
}

export interface GeoIPHit {
  ip: string;
  category: string;
  cidr: string;
}

export interface GeoDBHit {
  ip: string;
  cidr: string;
  record: string;
}

export interface GeoASNHit {
  ip: string;
  cidr: string;
  asn: number;
  organization: string;
}

export interface GeoQueryResponse {
  domain: string;
  /** "domain" | "ip" — present after geo query enhancement */
  inputType?: 'domain' | 'ip' | string;
  ips: string[];
  resolveSkipped: boolean;
  resolveError?: string;
  geosite: GeoSiteHit[];
  geoip: GeoIPHit[];
  metadb: GeoDBHit[];
  asn: GeoASNHit[];
}

export interface GeoReverseItem {
  value: string;
  type: string;
}

export interface GeoReverseResponse {
  file: string;
  category: string;
  items?: GeoReverseItem[];
  total: number;
  limit: number;
  offset: number;
  message?: string;
}

export interface GeoSearchResponse {
  file: string;
  field: string;
  keyword: string;
  items?: GeoEntryRow[];
  total: number;
  limit: number;
  offset: number;
  message?: string;
}

export interface GeoEntryRow {
  type: string;
  value: string;
  detail?: string;
}

export interface GeoUpdateItem {
  name: string;
  updated: boolean;
  error?: string;
}

export interface GeoUpdateResponse {
  items: GeoUpdateItem[];
}
