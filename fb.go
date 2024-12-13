package main

import (
	"FBCrawler/FBCrawler"
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

func callbackHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("/callback called...")

	cb, err := webhook.ParseRequest(channelSecret, req)
	if err != nil {
		log.Printf("Cannot parse request: %+v\n", err)
		if errors.Is(err, webhook.ErrInvalidSignature) {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	log.Println("Handling events...")
	for _, event := range cb.Events {
		log.Printf("/callback called%+v...\n", event)

		switch e := event.(type) {
		case webhook.MessageEvent:
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				usermessage := message.Text
				// usermessage will be "add ..." or "delete ..."
				// parse the message and update keywords
				// e.g., "add iPhone" -> add "iPhone" to keywords
				// e.g., "delete iPhone" -> delete "iPhone" from keywords
				// e.g., "list" -> list all keywords
				// e.g., "clear" -> clear all keywords
				
				parts := strings.Fields(usermessage)
				if len(parts) < 1 {
					http.Error(w, "Invalid message format", http.StatusBadRequest)
					return
				}

				command := parts[0]
				switch command {
				case "add":
					if len(parts) < 2 {
						http.Error(w, "Missing keyword to add", http.StatusBadRequest)
						return
					}
					newKeyword := parts[1]
					keywordsMutex.Lock()
					keywords = append(keywords, newKeyword)
					keywordsMutex.Unlock()
					fmt.Fprintf(w, "Keyword added: %s\n", newKeyword)

				case "delete":
					if len(parts) < 2 {
						http.Error(w, "Missing keyword to delete", http.StatusBadRequest)
						return
					}
					keywordToDelete := parts[1]
					keywordsMutex.Lock()
					for i, keyword := range keywords {
						if keyword == keywordToDelete {
							keywords = append(keywords[:i], keywords[i+1:]...)
							break
						}
					}
					keywordsMutex.Unlock()
					fmt.Fprintf(w, "Keyword deleted: %s\n", keywordToDelete)

				case "list":
					keywordsMutex.Lock()
					fmt.Fprintf(w, "Keywords: %v\n", keywords)
					keywordsMutex.Unlock()

				case "clear":
					keywordsMutex.Lock()
					keywords = []string{}
					keywordsMutex.Unlock()
					fmt.Fprintf(w, "All keywords cleared\n")

				default:
					http.Error(w, "Unknown command", http.StatusBadRequest)
				}
			}
		default:
			log.Printf("Unsupported message: %T\n", event)
		}
	}
}

func startAPIServer() {
	http.HandleFunc("/keywords", updateKeywordsHandler)
	http.HandleFunc("/callback", callbackHandler)

	go func() {
		log.Println("Starting API server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("API server failed:", err)
		}
	}()
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
		"冰箱", "手錶", "手套", "麻將",
		"GeForce", "BTS", "airpods", "二手",
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
			fmt.Println(post)
		}
		fmt.Println("Waiting for 10 minutes...")
		time.Sleep(10 * time.Minute)
	}
}
