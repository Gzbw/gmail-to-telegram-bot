package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	gmailAPI "google.golang.org/api/gmail/v1"

	"github.com/Gzbw/gmail-to-telegram-bot/internal/gmail"
	"github.com/Gzbw/gmail-to-telegram-bot/internal/telegram"
)


const sentFile = "sent.json"

// Загружает уже отправленные письма
func loadSent() map[string]bool {
	sent := make(map[string]bool)
	f, err := os.Open(sentFile)
	if err != nil {
		return sent
	}
	defer f.Close()
	var ids []string
	if err := json.NewDecoder(f).Decode(&ids); err != nil {
		return sent
	}
	for _, id := range ids {
		sent[id] = true
	}
	return sent
}

// Сохраняет отправленные письма
func saveSent(sent map[string]bool) {
	var ids []string
	for id := range sent {
		ids = append(ids, id)
	}
	f, err := os.Create(sentFile)
	if err != nil {
		log.Println("Ошибка сохранения sent.json:", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(ids)
}

func main() {
	apiToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if apiToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан")
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		log.Fatal("TELEGRAM_CHAT_ID не задан")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatal("TELEGRAM_CHAT_ID должен быть int64")
	}

	bot, err := telegram.NewBot(apiToken)
	if err != nil {
		log.Fatal(err)
	}

	srv := gmail.NewClient()
	log.Println("Бот запущен. Проверка новых писем каждые 30 секунд...")

	sent := loadSent() // загружаем уже отправленные письма

	for {
		res, err := srv.Users.Messages.List("me").LabelIds("INBOX", "UNREAD").MaxResults(10).Do()
		if err != nil {
			log.Println("Ошибка получения писем:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		newMessages := false

		for _, m := range res.Messages {
			if sent[m.Id] {
				continue
			}

			from, subject, body, err := gmail.GetMessageText(srv, m.Id)
			if err != nil {
				log.Println("Ошибка чтения письма:", err)
				continue
			}

			text := fmt.Sprintf("📧 От: %s\n📝 Тема: %s\n\n%s", from, subject, body)

			// Формируем ссылку на письмо в Gmail
			gmailLink := fmt.Sprintf("https://mail.google.com/mail/u/0/#inbox/%s", m.Id)

			// Отправляем сообщение с кнопкой "Открыть в Gmail"
			err = bot.SendMessageWithButton(chatID, text, gmailLink)
			if err != nil {
				log.Println("Ошибка отправки в Telegram:", err)
				continue
			}

			// Помечаем письмо как отправленное
			sent[m.Id] = true
			saveSent(sent)

			// Помечаем письмо как прочитанное
			_, err = srv.Users.Messages.Modify("me", m.Id, &gmailAPI.ModifyMessageRequest{
				RemoveLabelIds: []string{"UNREAD"},
			}).Do()
			if err != nil {
				log.Println("Ошибка пометки письма как прочитанного:", err)
			}

			newMessages = true
			log.Println("Письмо отправлено:", subject)
		}

		if !newMessages {
			log.Println("Новых писем нет")
		}

		time.Sleep(30 * time.Second)
	}
}
