package main

import (
	"flag"
	"fmt"
	"log"
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

func fbCrawler(account, password, groupID string, postLimit int) {
	fmt.Println("Account:", account)
	fmt.Println("Password:", password)
	fmt.Println("Group ID:", groupID)
	fmt.Println("Post Limit:", postLimit)
}

func main() {
	account, password, groupID, postLimit := parseArgs()
	fbCrawler(account, password, groupID, postLimit)
}