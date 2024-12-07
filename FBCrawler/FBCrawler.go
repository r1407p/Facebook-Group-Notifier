package FBCrawler
import (
	"fmt"
	"log"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)
type FBCrawler struct {
	Account   string
	Password  string
	GroupIDs  []string
	PostLimit int
	driver	selenium.WebDriver
}

func NewFBCrawler(account, password string, groupIDs []string, postLimit int) *FBCrawler {
	opts := []selenium.ServiceOption{}
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	chromeCaps := chrome.Capabilities{
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-notifications",
			"--start-maximized",
			"--headless",
		},
	}
	caps.AddChrome(chromeCaps)
	driver, err := initializeDriver(opts, caps)
	if err != nil {
		log.Fatal("Failed to initialize driver:", err)
	}
	defer driver.Quit()

	return &FBCrawler{
		Account:   account,
		Password:  password,
		GroupIDs:  groupIDs,
		PostLimit: postLimit,
	}
}

func initializeDriver(opts []selenium.ServiceOption, caps selenium.Capabilities) (selenium.WebDriver, error) {
	service, err := selenium.NewChromeDriverService("chromedriver", 9515, opts...)
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