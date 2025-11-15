package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Credentials contains cookie and token for mp api.
type Credentials struct {
	Cookie string
	Token  string
}

var httpClient = &http.Client{
	Timeout: 20 * time.Second,
}

// SearchResult maps mp searchbiz response items.
type SearchResult struct {
	Nickname string `json:"nickname"`
	Alias    string `json:"alias"`
	FakeID   string `json:"fakeid"`
	Province string `json:"province"`
	City     string `json:"city"`
}

type searchResponse struct {
	BaseResp struct {
		ErrMsg string `json:"err_msg"`
		Ret    int    `json:"ret"`
	} `json:"base_resp"`
	List  []SearchResult `json:"list"`
	Total int            `json:"total"`
}

// SearchAccounts searches mp accounts by name.
func SearchAccounts(ctx context.Context, cred Credentials, query string, begin int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("action", "search_biz")
	params.Set("token", cred.Token)
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")
	params.Set("begin", strconv.Itoa(begin))
	params.Set("count", "5")
	params.Set("query", query)
	params.Set("random", fmt.Sprintf("%f", time.Now().UTC().UnixNano()))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://mp.weixin.qq.com/cgi-bin/searchbiz?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", cred.Cookie)
	req.Header.Set("Referer", "https://mp.weixin.qq.com/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searchbiz status %d: %s", resp.StatusCode, string(body))
	}

	var parsed searchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.BaseResp.Ret != 0 {
		return nil, fmt.Errorf("searchbiz ret %d err %s", parsed.BaseResp.Ret, parsed.BaseResp.ErrMsg)
	}

	return parsed.List, nil
}

// ArticleItem describes mp article metadata.
type ArticleItem struct {
	Aid        string `json:"aid"`
	AppMsgID   string `json:"appmsgid"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Digest     string `json:"digest"`
	Cover      string `json:"cover"`
	Link       string `json:"link"`
	CreateTime int64  `json:"create_time"`
}

type appmsgResponse struct {
	BaseResp struct {
		Ret    int    `json:"ret"`
		ErrMsg string `json:"err_msg"`
	} `json:"base_resp"`
	AppMsgList []ArticleItem `json:"app_msg_list"`
	TotalCount int           `json:"total_count"`
}

// FetchArticles pulls article list for fakeid starting at offset.
func FetchArticles(ctx context.Context, cred Credentials, fakeid string, offset int, count int) (*appmsgResponse, error) {
	params := url.Values{}
	params.Set("action", "list_ex")
	params.Set("token", cred.Token)
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")
	params.Set("random", fmt.Sprintf("%f", time.Now().UnixNano()))
	params.Set("begin", strconv.Itoa(offset))
	params.Set("count", strconv.Itoa(count))
	params.Set("type", "9")
	params.Set("query", "")
	params.Set("fakeid", fakeid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://mp.weixin.qq.com/cgi-bin/appmsg?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", cred.Cookie)
	req.Header.Set("Referer", "https://mp.weixin.qq.com/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("appmsg status %d: %s", resp.StatusCode, string(body))
	}
	var parsed appmsgResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.BaseResp.Ret != 0 {
		return nil, fmt.Errorf("appmsg ret %d err %s", parsed.BaseResp.Ret, parsed.BaseResp.ErrMsg)
	}
	return &parsed, nil
}
