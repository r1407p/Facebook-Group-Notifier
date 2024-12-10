package FBCrawler  // Change package name to FBCrawler

import (
	"fmt"
	"strings"
)

// PostInfo structure to hold post content and URL
type PostInfo struct {
	Content string
	URL     string
}

func (p PostInfo) String() string {
	return fmt.Sprintf("Content: %s\nURL: %s", p.Content, p.URL)
}
func (p PostInfo) ContainsKeyword(keyword string) bool {
	return strings.Contains(p.Content, keyword)
}