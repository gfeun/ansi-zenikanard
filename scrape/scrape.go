// Package scrape provides a function to scrape zenikanards from the gallery website using playwright
package scrape

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gfeun/ansi-zenikanard/zenikanard"
	"github.com/mxschmitt/playwright-go"
)

const (
	baseURL            = "https://theduckgallery.zenika.com/"
	zenikanardSelector = "#gallery img[src]"
)

// FetchZenikanards parses duck gallery web page and get all zenikanard html img
// it then iterates over each one and send them through the out channel
func FetchZenikanards(zenikanards *zenikanard.Zenikanards) error {
	zenikanards.Lock()
	defer zenikanards.Unlock()

	log.Info("Starting scraping of ", baseURL)

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Firefox.Launch()
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	if _, err = page.Goto(baseURL, playwright.PageGotoOptions{WaitUntil: playwright.String("networkidle")}); err != nil {
		log.Fatalf("could not goto: %v", err)
	}

	entries, err := page.QuerySelectorAll(zenikanardSelector)
	if err != nil {
		log.Fatalf("could not get entries: %v", err)
	}

	for _, entry := range entries {
		link, err := entry.GetAttribute("src")
		if err != nil {
			log.Fatalf("could not get img src attribute: %v", err)
		}

		name, err := entry.GetAttribute("alt")
		if err != nil {
			log.Fatalf("could not get img alt attribute: %v", err)
		}

		zenikanards.List = append(zenikanards.List, &zenikanard.Zenikanard{Name: name, URL: baseURL + strings.TrimLeft(link, "./")})
	}

	if err = browser.Close(); err != nil {
		log.Fatalf("could not close browser: %v", err)
	}
	if err = pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}

	log.Println("Scraped", baseURL, "successfully", " - Retrieved: ", len(zenikanards.List), "Zenikanards")
	return nil
}
