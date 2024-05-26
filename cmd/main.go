package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hotmonth/news-feed-bot/internal/bot"
	"github.com/Hotmonth/news-feed-bot/internal/botkit"
	"github.com/Hotmonth/news-feed-bot/internal/config"
	"github.com/Hotmonth/news-feed-bot/internal/fetcher"
	"github.com/Hotmonth/news-feed-bot/internal/notifier"
	"github.com/Hotmonth/news-feed-bot/internal/storage"
	"github.com/Hotmonth/news-feed-bot/internal/summary"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	botApi, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Printf("failed to create bot api: %v", err)
		return
	}

	db, err := sqlx.Connect("postgres", config.Get().DatabaseDSN)
	if err != nil {
		log.Printf("failed to connect to databes: %v", err)
		return
	}
	defer db.Close()

	var (
		articleStorage = storage.NewArticleStorage(db)
		sourceStorage  = storage.NewSourceStorage(db)
		fetcher        = fetcher.New(
			articleStorage,
			sourceStorage,
			config.Get().FetchInterval,
			config.Get().FilterKeywords,
		)
		notifier = notifier.New(
			articleStorage,
			summary.NewGeminiSummarizer(config.Get().GeminiAPIKey, config.Get().GeminiPrompt),
			botApi,
			config.Get().NotificationInterval,
			2*config.Get().FetchInterval,
			config.Get().TelegramChannelID,
		)
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	newBot := botkit.New(botApi)
	newBot.RegisterCmdView("start", bot.ViewCmdStart())

	go func(ctx context.Context) {
		if err := fetcher.Start(ctx); err != nil {
			log.Printf("[ERROR] failed to start fetcher: %v", err)
			return
		}
		log.Println("fetcher stopped")
	}(ctx)

	go func(ctx context.Context) {
		if err := notifier.Start(ctx); err != nil {
			log.Printf("[ERROR] failed to start notifier: %v", err)
			return
		}

		log.Println("notifier stopped")
	}(ctx)

	if err := newBot.Start(ctx); err != nil {
		log.Printf("[ERROR] failed to start bot: %v", err)
		return
	}

}
