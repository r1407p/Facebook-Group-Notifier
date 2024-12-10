package main

import (
	"flag"
	"fmt"
	"log"
	"time"
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
		"GeForce", "BTS", "airpods", "二手"}
	for _, keyword := range keywords {
		fbcrawler.AddKeyword(keyword)
	}
	for {
		fmt.Println("Scanning for new posts...")
		newPosts, err := fbcrawler.ScanGroupPostsWithTopK(5)
		if err != nil {
			log.Fatal("Failed to scan group posts:", err)
		}
		// for _, post := range newPosts {
		// 	fmt.Println(post)
		// }
		post_with_keywords := fbcrawler.FilterPosts(newPosts)
		for _, post := range post_with_keywords {
			fmt.Println(post)
		}
		fmt.Println("Waiting for 10 minutes...")
		time.Sleep(10 * time.Minute)
	}
	return
}
