package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Bot struct {
	API *tgbotapi.BotAPI
}

func NewBot(apiToken string) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		return nil, err
	}
	return &Bot{API: botAPI}, nil
}

// Обычное сообщение
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.API.Send(msg)
	return err
}

// Сообщение с кнопкой "Открыть в Gmail"
func (b *Bot) SendMessageWithButton(chatID int64, text, gmailLink string) error {
	msg := tgbotapi.NewMessage(chatID, text)

	button := tgbotapi.NewInlineKeyboardButtonURL("📧 Открыть в Gmail", gmailLink)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(button))
	msg.ReplyMarkup = keyboard

	_, err := b.API.Send(msg)
	return err
}
