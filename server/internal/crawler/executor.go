package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"

	"wechat2rss/internal/models"
	"wechat2rss/internal/wechat"
)

// Executor defines how a crawl task is executed.
type Executor interface {
	Execute(ctx context.Context, task *models.Task) error
}

// ArticleExecutor fetches articles via mp api and stores them.
type ArticleExecutor struct {
	db     *gorm.DB
	client *http.Client
}

func NewArticleExecutor(db *gorm.DB) *ArticleExecutor {
	return &ArticleExecutor{
		db: db,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (e *ArticleExecutor) Execute(ctx context.Context, task *models.Task) error {
	var account models.Account
	if err := e.db.Preload("Session").First(&account, "id = ?", task.AccountID).Error; err != nil {
		return fmt.Errorf("load account: %w", err)
	}
	if account.BizID == "" {
		return errors.New("account missing biz_id")
	}
	if account.Session == nil || account.Session.Status != models.SessionStatusActive {
		return errors.New("account session invalid")
	}

	cred := wechat.Credentials{
		Cookie: account.Session.Cookie,
		Token:  account.Session.Token,
	}

	offset := 0
	const batch = 5
	for {
		resp, err := wechat.FetchArticles(ctx, cred, account.BizID, offset, batch)
		if err != nil {
			return fmt.Errorf("fetch articles: %w", err)
		}
		if len(resp.AppMsgList) == 0 {
			break
		}
		for _, item := range resp.AppMsgList {
			if err := e.saveArticle(ctx, account.ID, item); err != nil {
				return err
			}
		}
		offset += len(resp.AppMsgList)
		if offset >= resp.TotalCount {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func (e *ArticleExecutor) saveArticle(ctx context.Context, accountID uint, item wechat.ArticleItem) error {
	var existing models.Article
	if err := e.db.First(&existing, "wechat_article_id = ?", item.Aid).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	content, err := e.fetchContent(ctx, item.Link)
	if err != nil {
		content = ""
	}
	published := time.Unix(item.CreateTime, 0)

	article := models.Article{
		AccountID:       accountID,
		WechatArticleID: item.Aid,
		Title:           item.Title,
		Summary:         item.Digest,
		ContentHTML:     content,
		RawURL:          item.Link,
		PublishedAt:     published,
	}
	return e.db.Create(&article).Error
}

func (e *ArticleExecutor) fetchContent(ctx context.Context, link string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 Wechat2RSS")
	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fetch content %d: %s", resp.StatusCode, string(body))
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	html, err := doc.Find("#js_content").Html()
	if err != nil {
		return "", err
	}
	return html, nil
}
