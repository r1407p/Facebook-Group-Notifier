package main

import (
	"flag"
	"fmt"
	"log"
	"FBCrawler/FBCrawler"
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


func main() {
	account, password, groupID, postLimit := parseArgs()

	fbcrawler := FBCrawler.NewFBCrawler(account, password, groupID, postLimit)
	fmt.Println(fbcrawler)

	if err := fbcrawler.LoginToFacebook(); err != nil {
		log.Fatal("Login failed:", err)
	}
	keywords := []string{
		"冰箱", "手錶", "手套", "麻將", 
		"GeForce", "BTS", "airpods"}
	for _, keyword := range keywords {
		fbcrawler.AddKeyword(keyword)
	}
	
	return
}
