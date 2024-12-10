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