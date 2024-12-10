package FBCrawler  // Change package name to FBCrawler

import "fmt"

// PostInfo structure to hold post content and URL
type PostInfo struct {
	Content string
	URL     string
}

func (p PostInfo) String() string {
	return fmt.Sprintf("Content: %s\nURL: %s", p.Content, p.URL)
}