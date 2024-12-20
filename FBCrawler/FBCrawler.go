package FBCrawler
import (
	"fmt"
	"log"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"time"
)
type FBCrawler struct {
	Account   string
	Password  string
	GroupID  string
	PostLimit int
	// keywords  []string
	viewedPosts [] PostInfo
	Driver	selenium.WebDriver
}

func NewFBCrawler(account string, password string, groupID string, postLimit int) *FBCrawler {
	opts := []selenium.ServiceOption{}
	caps := selenium.Capabilities{"browserName": "chrome"}
	chromeCaps := chrome.Capabilities{
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-notifications", // Block notifications
			"--start-maximized",
		},
	}
	caps.AddChrome(chromeCaps)

	// Start Chrome
	driver, err := initializeDriver(opts, caps)
	if err != nil {
		log.Fatal("Failed to initialize driver:", err)
	}

	return &FBCrawler{
		Account:   account,
		Password:  password,
		GroupID:   groupID,
		PostLimit: postLimit,
		// keywords:  []string{},
		viewedPosts: []PostInfo{},
		Driver:    driver,
	}
}

func initializeDriver(opts []selenium.ServiceOption, caps selenium.Capabilities) (selenium.WebDriver, error) {
	service, err := selenium.NewChromeDriverService("./chromedriver.exe", 9515, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start ChromeDriver: %v", err)
	}

	driver, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", 9515))
	if err != nil {
		service.Stop()
		return nil, fmt.Errorf("failed to create driver: %v", err)
	}

	return driver, nil
}

func (fbc *FBCrawler) LoginToFacebook() error {
	if err := fbc.Driver.Get("https://www.facebook.com"); err != nil {
		return err
	}

	time.Sleep(2 * time.Second) // Wait for page load

	// Login
	if email, err := fbc.Driver.FindElement(selenium.ByCSSSelector, "input[name='email']"); err == nil {
		email.SendKeys(fbc.Account)
	} else {
		return fmt.Errorf("couldn't find email input: %v", err)
	}

	if pass, err := fbc.Driver.FindElement(selenium.ByCSSSelector, "input[name='pass']"); err == nil {
		pass.SendKeys(fbc.Password)
	} else {
		return fmt.Errorf("couldn't find password input: %v", err)
	}

	if loginBtn, err := fbc.Driver.FindElement(selenium.ByCSSSelector, "button[name='login']"); err == nil {
		loginBtn.Click()
	} else {
		return fmt.Errorf("couldn't find login button: %v", err)
	}

	time.Sleep(30 * time.Second) // Wait for login
	return nil
}

// func (fbc *FBCrawler) AddKeyword(keyword string) {
// 	fbc.keywords = append(fbc.keywords, keyword)
// }

func (f *FBCrawler) ScanGroupPostsWithTopK(topK int) ([]PostInfo, error) {
	groupURL := fmt.Sprintf("https://www.facebook.com/groups/%s", f.GroupID)
	if err := f.Driver.Get(groupURL); err != nil {
		return nil, err
	}
	time.Sleep(5 * time.Second)

	f.clickNewPost()

	postsFound := 0
	lastPostCount := 0
	attempts := 0
	maxAttempts := 5

	var allPosts []PostInfo

	for postsFound < topK && attempts < maxAttempts {
		f.Driver.ExecuteScript("window.scrollTo(0, document.body.scrollHeight)", nil)
		f.expandPosts()

		time.Sleep(4 * time.Second)
		posts, _ := f.Driver.FindElements(selenium.ByCSSSelector, "div.x1yztbdb:not([aria-hidden='true'])")

		if len(posts) == lastPostCount {
			attempts++
		} else {
			lastPostCount = len(posts)
			attempts = 0
		}

		for _, post := range posts {
			text, err := post.Text()
			if err != nil {
				continue
			}

			postText := []rune(text)
			if len(postText) < 5 {
				continue
			}

			// Directly add the post to the list without filtering by keywords
			var postURL string
			if links, err := post.FindElements(selenium.ByCSSSelector, "a[href*='/groups/']"); err == nil && len(links) > 0 {
				if href, err := links[0].GetAttribute("href"); err == nil {
					postURL = href
				}
			}

			allPosts = append(allPosts, PostInfo{
				Content: text,
				URL:     postURL,
			})

			postsFound++

			if postsFound >= topK {
				break
			}
		}

		if postsFound >= topK {
			break
		}
	}
	newPosts := []PostInfo{}
	for _, post := range allPosts {
		if !f.hasViewedPost(post) {
			f.viewedPosts = append(f.viewedPosts, post)
			newPosts = append(newPosts, post)
		}
	}
	if len(f.viewedPosts) > 10 {
		f.viewedPosts = f.viewedPosts[len(f.viewedPosts)-10:]
	}
	return newPosts, nil
}

func (f *FBCrawler) hasViewedPost(post PostInfo) bool {
	for _, viewedPost := range f.viewedPosts {
		if viewedPost.Content == post.Content {
			return true
		}
	}
	return false
}

func (f *FBCrawler) expandPosts() {
    if seeMoreBtns, err := f.Driver.FindElements(selenium.ByXPATH, "//div[contains(text(),'查看更多')]"); err == nil {
        for _, btn := range seeMoreBtns {
            if err := btn.Click(); err == nil {
                time.Sleep(1 * time.Second)
            }
        }
    }
    time.Sleep(3 * time.Second)
}


func (f *FBCrawler) clickNewPost() {
    if expandbutton, err := f.Driver.FindElement(selenium.ByXPATH, "//span[contains(text(),'最相關')]"); err == nil {
        fmt.Println("找到最相關")
        expandbutton.Click()
    }
    time.Sleep(3 * time.Second)
    if newPostBtn, err := f.Driver.FindElement(selenium.ByXPATH, "//span[contains(text(),'新貼文')]"); err == nil {
        fmt.Println("找到新貼文")
        newPostBtn.Click()
    }
    time.Sleep(3 * time.Second)
}

func (f *FBCrawler) FilterPosts(posts []PostInfo, keywords []string) []PostInfo {
	var filteredPosts []PostInfo
	for _, post := range posts {
		for _, keyword := range keywords {
			keywordRunes := []rune(keyword)
			if len(keywordRunes) < 2 {
				continue
			}
			if post.ContainsKeyword(string(keywordRunes)) {
				filteredPosts = append(filteredPosts, post)
				break
			}
		}
	}
	return filteredPosts
}
