package main

import (
	"FACEBOOK-GROUP-NOTIFIER/FBCrawler"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"bytes"
	"errors"
	"sync"
	"time"
	"strings"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

var keywordsMutex sync.Mutex
var keywords []string 
var (
	bot            *messaging_api.MessagingApiAPI
	channelSecret  string
	err            error
	notifyToken    string
)

func parseArgs() (string, string, string, int) {
	account := flag.String("account", "", "Facebook account")
	password := flag.String("password", "", "Facebook password")
	groupID := flag.String("group", "817620721658179", "Facebook group ID")
	postLimit := flag.Int("limit", 10, "Number of posts to scan")
	flag.Parse()

	if *account == "" || *password == "" {
		log.Fatal("Account and password are required. Use -account and -password flags")
	}

	return *account, *password, *groupID, *postLimit
}

func updateKeywordsHandler(w http.ResponseWriter, r *http.Request) {
	keywordsMutex.Lock()
	defer keywordsMutex.Unlock()

	switch r.Method {
	case http.MethodPost: // Add a new keyword
		newKeyword := r.URL.Query().Get("keyword")
		if newKeyword != "" {
			// Avoid adding duplicates
			for _, existingKeyword := range keywords {
				if existingKeyword == newKeyword {
					http.Error(w, "Keyword already exists", http.StatusBadRequest)
					return
				}
			}
			keywords = append(keywords, newKeyword)
			fmt.Fprintf(w, "Keyword added: %s\n", newKeyword)
		} else {
			http.Error(w, "Missing 'keyword' parameter", http.StatusBadRequest)
		}

	case http.MethodDelete: // Delete a keyword
		keywordToDelete := r.URL.Query().Get("keyword")
		if keywordToDelete != "" {
			// Find and remove the keyword
			for i, keyword := range keywords {
				if keyword == keywordToDelete {
					keywords = append(keywords[:i], keywords[i+1:]...)
					fmt.Fprintf(w, "Keyword deleted: %s\n", keywordToDelete)
					return
				}
			}
			http.Error(w, "Keyword not found", http.StatusNotFound)
		} else {
			http.Error(w, "Missing 'keyword' parameter", http.StatusBadRequest)
		}

	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func lineNotifyMessage(token, msg string) error {
	apiUrl := "https://notify-api.line.me/api/notify"
	data := url.Values{}
	data.Set("message", msg)
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("status code is not 200")
	}
	return nil
}

func callbackHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("/callback called...")

	cb, err := webhook.ParseRequest(channelSecret, req)
	if err != nil {
		log.Printf("Cannot parse request: %+v\n", err)
		if errors.Is(err, webhook.ErrInvalidSignature) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	log.Println("Handling events...")
	for _, event := range cb.Events {
		log.Printf("Event received: %+v\n", event)

		switch e := event.(type) {
		case webhook.MessageEvent:
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				usermessage := message.Text

				// Parse the message and handle commands
				parts := strings.Fields(usermessage)
				if len(parts) < 1 {
					log.Println("Invalid message format")
					replyMessage(e.ReplyToken, "Invalid message format. Use commands like 'add <keyword>' or 'delete <keyword>'.")
					continue
				}

				command := parts[0]
				switch command {
				case "add":
					if len(parts) < 2 {
						log.Println("Missing keyword to add")
						replyMessage(e.ReplyToken, "Missing keyword to add. Use: 'add <keyword>'.")
						continue
					}
					newKeyword := parts[1]
					keywordsMutex.Lock()
					keywords = append(keywords, newKeyword)
					keywordsMutex.Unlock()
					log.Printf("Keyword added: %s\n", newKeyword)
					replyMessage(e.ReplyToken, "Keyword added: "+newKeyword)

				case "delete":
					if len(parts) < 2 {
						log.Println("Missing keyword to delete")
						replyMessage(e.ReplyToken, "Missing keyword to delete. Use: 'delete <keyword>'.")
						continue
					}
					keywordToDelete := parts[1]
					keywordsMutex.Lock()
					deleted := false
					for i, keyword := range keywords {
						if keyword == keywordToDelete {
							keywords = append(keywords[:i], keywords[i+1:]...)
							deleted = true
							break
						}
					}
					keywordsMutex.Unlock()
					if deleted {
						log.Printf("Keyword deleted: %s\n", keywordToDelete)
						replyMessage(e.ReplyToken, "Keyword deleted: "+keywordToDelete)
					} else {
						log.Printf("Keyword not found: %s\n", keywordToDelete)
						replyMessage(e.ReplyToken, "Keyword not found: "+keywordToDelete)
					}

				case "list":
					keywordsMutex.Lock()
					log.Printf("Keywords listed: %v\n", keywords)
					replyMessage(e.ReplyToken, fmt.Sprintf("Keywords: %v", keywords))
					keywordsMutex.Unlock()

				case "clear":
					keywordsMutex.Lock()
					keywords = []string{}
					keywordsMutex.Unlock()
					log.Println("All keywords cleared")
					replyMessage(e.ReplyToken, "All keywords cleared.")

				default:
					log.Printf("Unknown command: %s\n", command)
					replyMessage(e.ReplyToken, "Unknown command. Supported commands: add, delete, list, clear.")
				}
			}
		default:
			log.Printf("Unsupported event type: %T\n", event)
		}
	}
}

// replyMessage sends a text reply to the user
func replyMessage(replyToken, message string) {
	replyRequest := &messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message,
			},
		},
	}

	if _, err := bot.ReplyMessage(replyRequest); err != nil {
		log.Printf("Failed to send reply: %v\n", err)
	} else {
		log.Println("Sent text reply.")
	}
}

func startAPIServer() {
	http.HandleFunc("/keywords", updateKeywordsHandler)
	http.HandleFunc("/callback", callbackHandler)

	go func() {
		log.Println("http://localhost:" + "8080" + "/")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("API server failed:", err)
		}
	}()
	// fmt.Println("http://localhost:" + "8080" + "/")
	// if err := http.ListenAndServe(":"+"8080", nil); err != nil {
	// 	log.Fatal(err)
	// }
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	channelSecret = os.Getenv("ChannelSecret")
	bot, err = messaging_api.NewMessagingApiAPI(
		os.Getenv("ChannelAccessToken"),
	)
	if err != nil {
		log.Fatal(err)
	}


	notifyToken = os.Getenv("LineNotifyToken")
	if notifyToken == "" {
		log.Fatal("NotifyToken is not set")
	}

	// Initial keywords
	keywords = []string{
		"冰箱", "手錶", "手套", "麻將", "工讀",
		"GeForce", "BTS", "airpods", "二手", "電影票",
	}
	startAPIServer()
	account, password, groupID, postLimit := parseArgs()

	fbcrawler := FBCrawler.NewFBCrawler(account, password, groupID, postLimit)

	if err := fbcrawler.LoginToFacebook(); err != nil {
		log.Fatal("Login failed:", err)
	}

	for {
		fmt.Println("Scanning for new posts...")
		newPosts, err := fbcrawler.ScanGroupPostsWithTopK(5)
		if err != nil {
			log.Fatal("Failed to scan group posts:", err)
		}

		keywordsMutex.Lock()
		post_with_keywords := fbcrawler.FilterPosts(newPosts, keywords)
		keywordsMutex.Unlock()

		for _, post := range post_with_keywords {
			message := fmt.Sprintf("New post: %s\n%s", post.Content, post.URL)
			fmt.Println(message)
			if err := lineNotifyMessage(notifyToken, message); err != nil {
				log.Println("Failed to send LINE notify:", err)
			}
		}
		fmt.Println("Waiting for 10 minutes...")
		time.Sleep(10 * time.Minute)
	}
}
