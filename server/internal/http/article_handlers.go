package http

import (
	"encoding/xml"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wechat2rss/internal/models"
)

func (s *Server) handleListArticles(c *gin.Context) {
	account, err := s.findAccount(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "account not found")
		return
	}

	var articles []models.Article
	if err := s.db.Where("account_id = ?", account.ID).
		Order("published_at desc").
		Limit(50).
		Find(&articles).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load articles")
		return
	}

	respondOK(c, apiData{
		"account":  toAccountView(account),
		"articles": articles,
	})
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Content     string `xml:"encoded,omitempty"`
}

func (s *Server) handleFeed(c *gin.Context) {
	account, err := s.findAccount(c.Param("id"))
	if err != nil {
		c.String(http.StatusNotFound, "account not found")
		return
	}

	var articles []models.Article
	if err := s.db.Where("account_id = ?", account.ID).
		Order("published_at desc").
		Limit(50).
		Find(&articles).Error; err != nil {
		c.String(http.StatusInternalServerError, "query error")
		return
	}

	items := make([]rssItem, 0, len(articles))
	for _, article := range articles {
		items = append(items, rssItem{
			Title:       article.Title,
			Link:        article.RawURL,
			Description: article.Summary,
			PubDate:     article.PublishedAt.Format(time.RFC1123Z),
			GUID:        article.WechatArticleID,
			Content:     article.ContentHTML,
		})
	}

	host := c.Request.Host
	channel := rssChannel{
		Title:         account.Name,
		Link:          "https://" + host + "/feed/" + strconv.Itoa(int(account.ID)),
		Description:   account.Alias,
		LastBuildDate: time.Now().Format(time.RFC1123Z),
		Items:         items,
	}
	feed := rssFeed{
		Version: "2.0",
		Channel: channel,
	}

	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	enc := xml.NewEncoder(c.Writer)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		c.String(http.StatusInternalServerError, "encode error")
		return
	}
}
