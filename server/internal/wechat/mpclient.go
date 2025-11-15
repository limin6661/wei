package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	qrEndpoint   = "https://mp.weixin.qq.com/cgi-bin/scanloginqrcode"
	askEndpoint  = "https://mp.weixin.qq.com/cgi-bin/scanloginqrcode"
	refererURL   = "https://mp.weixin.qq.com/"
	loginTimeout = 120 * time.Second
)

// MPClient interacts with mp.weixin.qq.com login endpoints.
type MPClient struct {
	client *http.Client
	jar    http.CookieJar
}

// NewMPClient constructs a client with cookie jar.
func NewMPClient() (*MPClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &MPClient{
		client: &http.Client{
			Jar: jar,
		},
		jar: jar,
	}, nil
}

// QRCode holds uuid and binary data.
type QRCode struct {
	UUID string
	Data []byte
}

// FetchQRCode requests a new QR code image and uuid.
func (c *MPClient) FetchQRCode(ctx context.Context) (*QRCode, error) {
	random := rand.Float64()
	u := fmt.Sprintf("%s?action=getqrcode&random=%f", qrEndpoint, random)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", refererURL)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	uuid := fetchCookie(resp.Cookies(), "uuid")
	if uuid == "" {
		// sometimes uuid stored in jar
		uuid = c.cookieValue("uuid")
	}
	if uuid == "" {
		return nil, fmt.Errorf("uuid not found in cookies")
	}

	return &QRCode{UUID: uuid, Data: body}, nil
}

type askResponse struct {
	Status      int      `json:"status"`
	BaseResp    baseResp `json:"base_resp"`
	AuthURL     string   `json:"redirect_url"`
	ExpiredTime int64    `json:"expired_time"`
}

type baseResp struct {
	ErrMsg string `json:"err_msg"`
}

// LoginStatus describes scanning state.
type LoginStatus struct {
	State       string
	RedirectURL string
	Expired     bool
}

// AskStatus checks QR code scanning status.
func (c *MPClient) AskStatus(ctx context.Context, uuid string) (*LoginStatus, error) {
	params := url.Values{}
	params.Set("action", "ask")
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")
	params.Set("random", fmt.Sprintf("%f", rand.Float64()))
	params.Set("uuid", uuid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?%s", askEndpoint, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", refererURL)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status ask error %d: %s", resp.StatusCode, string(body))
	}
	var decoded askResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}

	state := "waiting"
	switch decoded.Status {
	case 0:
		state = "waiting"
	case 1:
		state = "scanned"
	case 2:
		state = "authorized"
	case 3:
		state = "expired"
	}
	return &LoginStatus{
		State:       state,
		RedirectURL: decoded.AuthURL,
		Expired:     decoded.Status == 3,
	}, nil
}

// FinalizeLogin follows redirect url and extracts cookies + token.
func (c *MPClient) FinalizeLogin(ctx context.Context, redirectURL string) (string, string, error) {
	if redirectURL == "" {
		return "", "", fmt.Errorf("redirect url empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Referer", refererURL)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	token := parseToken(redirectURL)
	if token == "" {
		if resp.Request != nil && resp.Request.URL != nil {
			token = parseToken(resp.Request.URL.String())
		}
	}
	if token == "" {
		return "", "", fmt.Errorf("token not found")
	}
	cookies := c.serializeCookies("https://mp.weixin.qq.com")
	return cookies, token, nil
}

func (c *MPClient) serializeCookies(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	cookies := c.jar.Cookies(u)
	var sb strings.Builder
	for i, ck := range cookies {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(ck.Name)
		sb.WriteString("=")
		sb.WriteString(ck.Value)
	}
	return sb.String()
}

func (c *MPClient) cookieValue(name string) string {
	u, _ := url.Parse(refererURL)
	for _, ck := range c.jar.Cookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func fetchCookie(cookies []*http.Cookie, name string) string {
	for _, ck := range cookies {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func parseToken(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if token := u.Query().Get("token"); token != "" {
		return token
	}
	fragments := strings.Split(raw, "token=")
	if len(fragments) > 1 {
		tokenPart := fragments[1]
		for i := 0; i < len(tokenPart); i++ {
			if !isTokenRune(rune(tokenPart[i])) {
				return tokenPart[:i]
			}
		}
		return tokenPart
	}
	return ""
}

func isTokenRune(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
