package main

import (
	"FBCrawler/FBCrawler"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var keywordsMutex sync.Mutex
var keywords []string 

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

func startAPIServer() {
	http.HandleFunc("/keywords", updateKeywordsHandler)
	go func() {
		log.Println("Starting API server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("API server failed:", err)
		}
	}()
}

func main() {
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
