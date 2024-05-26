package fetcher

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Hotmonth/news-feed-bot/internal/model"
	"github.com/Hotmonth/news-feed-bot/internal/source"
	"go.tomakado.io/containers/set"
)

type ArticleStorage interface {
	Store(ctx context.Context, article model.Article) error
}

type SourceProvider interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

type Source interface {
	ID() int64
	Name() string
	Fetch(ctx context.Context) ([]model.Item, error)
}

type Fetcher struct {
	articles ArticleStorage
	source   SourceProvider

	fetchInterval  time.Duration
	filterKeywords []string
}

func New(
	articleStorage ArticleStorage,
	sourceProvider SourceProvider,
	fetchInterval time.Duration,
	filterKeywords []string,
) *Fetcher {
	return &Fetcher{
		articles:       articleStorage,
		source:         sourceProvider,
		fetchInterval:  fetchInterval,
		filterKeywords: filterKeywords,
	}
}

func (f *Fetcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	if err := f.Fetch(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.source.Sources(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)

		rssSource := source.NewRSSSourceFromModel(src)

		go func(source Source) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("failed to fetch items from source %q: %v", source.Name(), err)
				return
			}

			if err := f.processItems(ctx, rssSource, items); err != nil {
				log.Printf("failed to process items from source %q: %v", source.Name(), err)
				return
			}
		}(rssSource)
	}
	wg.Wait()

	return nil

}

func (f *Fetcher) processItems(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		item.Date = item.Date.UTC()

		if f.itemShouldBeSkipped(item) {
			continue
		}

		if err := f.articles.Store(ctx, model.Article{
			SourceID:    source.ID(),
			Title:       item.Title,
			Link:        item.Link,
			Summary:     item.Summary,
			PublishedAt: item.Date,
		}); err != nil {
			return err
		}

	}
	return nil
}

func (f *Fetcher) itemShouldBeSkipped(item model.Item) bool {
	categoriesSet := set.New(item.Categories...)

	for _, keyword := range f.filterKeywords {
		titleContainsKeyword := strings.Contains(strings.ToLower(item.Title), keyword)
		if categoriesSet.Contains(keyword) || titleContainsKeyword {
			return true
		}
	}
	return false
}