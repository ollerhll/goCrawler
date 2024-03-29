package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

var regExpLink *regexp.Regexp = regexp.MustCompile(`(?:src)|(?:href)="([^"\s]+)"`)

//crawlers take a domain and crawl it, following every
//internal link, and printing all links and static assets
type Crawler struct {
	urls       chan string
	outputChan chan []string
	domain     string
	Filter
	sync.WaitGroup
}

func (crawler *Crawler) crawl(url string) {
	defer crawler.Done()
	messages := make([]string, 0)
	defer func() {
		crawler.outputChan <- messages
	}()
	messages = append(messages, fmt.Sprintf("=======================================================\n\nCrawling page: %s", url))
	response, err := http.Get(url)
	if err != nil {
		errorMessage := fmt.Sprintf("Error crawling %s: %v", url, err)
		messages = append(messages, errorMessage)
		return
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("Error crawling %s: %v", url, err)
		messages = append(messages, errorMessage)
		return
	}
	bodyStr := string(body)
	messages = crawler.extractURLs(url, bodyStr, messages)
}

func (crawler *Crawler) extractURLs(crawledURL, responseBody string, messages []string) []string {
	if foundURLs := regExpLink.FindAllStringSubmatch(responseBody, -1); foundURLs != nil {
		url, err := url.Parse(crawledURL)
		if err != nil {
			errorMessage := fmt.Sprintf("	Error crawling %s: %v", url, err)
			messages = append(messages, errorMessage)
			return messages
		}
		for _, foundURL := range foundURLs {
			rawLink := foundURL[1]
			if rawLink != "" {
				parsedLink, err := url.Parse(rawLink)
				if err != nil {
					errorMessage := fmt.Sprintf("	Error crawling %s on discovered link %s: %v", url, rawLink, err)
					messages = append(messages, errorMessage)
					return messages
				}
				discoveredURL := fmt.Sprintf("	  Discovered link %s on page %s", rawLink, crawledURL)
				messages = append(messages, discoveredURL)
				if parsedLink.IsAbs() {
					messages = append(messages, crawler.addURL(rawLink))
				} else {
					crawler.addURL(url.ResolveReference(parsedLink).String())
				}
			}
		}
	}
	return messages
}

func (crawler *Crawler) addURL(url string) string {
	if crawler.ShouldCrawl(url) {
		crawler.UpdateCrawledURLs(url)
		crawler.Add(1)
		crawler.urls <- url
		return ""
	}
	return ""
}

func InitCrawler(domain string) *Crawler {
	c := Crawler{
		make(chan string),
		make(chan []string),
		domain,
		Filter{
			make([]FilterFunction, 0),
			make(map[string]bool),
			sync.RWMutex{},
		},
		sync.WaitGroup{},
	}
	c.AddFilterFunction(func(url string) bool {
		return strings.HasPrefix(url, domain)
	})
	return &c
}

func (crawler *Crawler) StartCrawling() {
	go func() {
		for output := range crawler.outputChan {
			for _, message := range output {
				fmt.Println(message)
			}
		}
	}()

	go func() {
		for url := range crawler.urls {
			go crawler.crawl(url)
		}
	}()

	crawler.Add(1)
	crawler.urls <- crawler.domain

	crawler.Wait()

	crawler.Stop()
}

func (crawler *Crawler) Stop() {
	close(crawler.urls)
	close(crawler.outputChan)
}
