package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailAPI "google.golang.org/api/gmail/v1"
)

// NewClient создаёт Gmail клиент
func NewClient() *gmailAPI.Service {
	ctx := context.Background()

	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Не удалось прочитать credentials.json: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmailAPI.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Не удалось создать конфигурацию OAuth2: %v", err)
	}

	tokenFile := "token.json"
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}

	client := config.Client(ctx, tok)
	srv, err := gmailAPI.New(client)
	if err != nil {
		log.Fatalf("Не удалось создать Gmail сервис: %v", err)
	}
	return srv
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var tok oauth2.Token
	err = json.NewDecoder(f).Decode(&tok)
	return &tok, err
}

func saveToken(path string, token *oauth2.Token) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Не удалось создать файл токена: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Printf("Откройте в браузере этот URL и вставьте код:\n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Не удалось прочитать код: %v", err)
	}

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Не удалось получить токен: %v", err)
	}
	return tok
}

// htmlToText конвертирует HTML в чистый текст
func htmlToText(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}

	var f func(*html.Node)
	var sb strings.Builder
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			sb.WriteString("\n")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	lines := strings.Split(sb.String(), "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

// GetMessageText возвращает From, Subject и тело письма (чистый текст)
func GetMessageText(srv *gmailAPI.Service, msgID string) (from, subject, body string, err error) {
	msg, err := srv.Users.Messages.Get("me", msgID).Format("full").Do()
	if err != nil {
		return "", "", "", err
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "From":
			from = h.Value
		case "Subject":
			subject = h.Value
		}
	}

	if msg.Payload.MimeType == "text/plain" {
		data, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		body = string(data)
	} else if msg.Payload.MimeType == "text/html" {
		data, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		body = htmlToText(string(data))
	} else if strings.HasPrefix(msg.Payload.MimeType, "multipart/") {
		for _, part := range msg.Payload.Parts {
			if part.MimeType == "text/plain" && part.Body.Data != "" {
				data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
				body = string(data)
				break
			}
			if part.MimeType == "text/html" && part.Body.Data != "" && body == "" {
				data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
				body = htmlToText(string(data))
			}
		}
	}

	if len(body) > 4000 {
		body = body[:4000] + "..."
	}

	body = strings.TrimSpace(body)
	return from, subject, body, nil
}
