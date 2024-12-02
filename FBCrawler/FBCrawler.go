package FBCrawler

type FBCrawler struct {
	Account   string
	Password  string
	GroupIDs  []string
	PostLimit int
}

func NewFBCrawler(account, password string, groupIDs []string, postLimit int) *FBCrawler {
	return &FBCrawler{
		Account:   account,
		Password:  password,
		GroupIDs:  groupIDs,
		PostLimit: postLimit,
	}
}