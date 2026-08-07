package ipgeo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultURL     = "https://ipwho.is/{ip}"
	DefaultTimeout = 5 * time.Second
	maxBodyBytes   = 1 << 20
)

type Flag struct {
	Img          string `json:"img,omitempty"`
	Emoji        string `json:"emoji,omitempty"`
	EmojiUnicode string `json:"emoji_unicode,omitempty"`
}

type Result struct {
	IP            string  `json:"ip"`
	Continent     string  `json:"continent,omitempty"`
	ContinentCode string  `json:"continentCode,omitempty"`
	Country       string  `json:"country,omitempty"`
	CountryCode   string  `json:"countryCode,omitempty"`
	Region        string  `json:"region,omitempty"`
	RegionCode    string  `json:"regionCode,omitempty"`
	City          string  `json:"city,omitempty"`
	Postal        string  `json:"postal,omitempty"`
	Flag          Flag    `json:"flag,omitempty"`
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	ASN           string  `json:"asn,omitempty"`
	Organization  string  `json:"organization,omitempty"`
	ISP           string  `json:"isp,omitempty"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultURL
	}
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" || (u.Scheme != "https" && !isLocalHTTP(u)) {
		return nil, errors.New("IP geo URL must use HTTPS")
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Client{baseURL: u.String(), http: &http.Client{Timeout: timeout}}, nil
}

func (c *Client) Lookup(ctx context.Context, ip net.IP) (Result, error) {
	if c == nil || c.http == nil {
		return Result{}, errors.New("IP geo client unavailable")
	}
	if ip == nil {
		return Result{}, errors.New("invalid IP")
	}
	endpoint, err := c.endpoint(ip.String())
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, errors.New("create IP geo request failed")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return Result{}, errors.New("IP geo request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("IP geo provider returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		return Result{}, errors.New("IP geo response is too large or unreadable")
	}
	var payload struct {
		Success       bool    `json:"success"`
		Message       string  `json:"message"`
		IP            string  `json:"ip"`
		Continent     string  `json:"continent"`
		ContinentCode string  `json:"continent_code"`
		Country       string  `json:"country"`
		CountryCode   string  `json:"country_code"`
		Region        string  `json:"region"`
		RegionCode    string  `json:"region_code"`
		City          string  `json:"city"`
		Postal        string  `json:"postal"`
		Flag          Flag    `json:"flag"`
		Latitude      float64 `json:"latitude"`
		Longitude     float64 `json:"longitude"`
		Connection    struct {
			ASN          json.RawMessage `json:"asn"`
			Organization string          `json:"org"`
			ISP          string          `json:"isp"`
		} `json:"connection"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Result{}, errors.New("invalid IP geo response")
	}
	if !payload.Success {
		if payload.Message != "" {
			return Result{}, fmt.Errorf("IP geo provider rejected query: %s", payload.Message)
		}
		return Result{}, errors.New("IP geo provider rejected query")
	}
	return Result{
		IP:            firstNonEmpty(payload.IP, ip.String()),
		Continent:     payload.Continent,
		ContinentCode: payload.ContinentCode,
		Country:       payload.Country,
		CountryCode:   payload.CountryCode,
		Region:        payload.Region,
		RegionCode:    payload.RegionCode,
		City:          payload.City,
		Postal:        payload.Postal,
		Flag:          payload.Flag,
		Latitude:      payload.Latitude,
		Longitude:     payload.Longitude,
		ASN:           rawString(payload.Connection.ASN),
		Organization:  payload.Connection.Organization,
		ISP:           payload.Connection.ISP,
	}, nil
}

func (c *Client) endpoint(ip string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", errors.New("invalid IP geo URL")
	}
	encoded := url.PathEscape(ip)
	if strings.Contains(u.Path, "{ip}") {
		u.Path = strings.ReplaceAll(u.Path, "{ip}", encoded)
		return u.String(), nil
	}
	if strings.Contains(u.RawQuery, "{ip}") {
		u.RawQuery = strings.ReplaceAll(u.RawQuery, "{ip}", url.QueryEscape(ip))
		return u.String(), nil
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = strings.TrimRight(u.Path, "/") + "/" + encoded
		return u.String(), nil
	}
	q := u.Query()
	q.Set("ip", ip)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var number json.Number
	if json.Unmarshal(raw, &number) == nil {
		return number.String()
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isLocalHTTP(u *url.URL) bool {
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}
