package bot

import (
	"context"

	"github.com/Hotmonth/news-feed-bot/internal/botkit"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ViewCmdStart() botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		if _, err := bot.Send(tgbotapi.NewMessage(
			update.FromChat().ID,
			"Hello! I'm a news feed bot. I can help you to get news from your favorite sources.",
		)); err != nil {
			return err
		}
		return nil
	}
}
