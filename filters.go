package main

import "sync"

type FilterFunction func(string) bool

//filters hold both a list of predicates to filter URLs by
//and a concurrent (by virtue of the RWMutex) map
//containing URLs that have already been crawled
type Filter struct {
	filterFunctions []FilterFunction
	crawledURLs     map[string]bool
	sync.RWMutex
}

func (filter *Filter) AddFilterFunction(filterFunc FilterFunction) {
	filter.filterFunctions = append(filter.filterFunctions, filterFunc)
	return
}

func (filter *Filter) ShouldCrawl(url string) bool {
	for _, function := range filter.filterFunctions {
		if !function(url) {
			return false
		}
	}
	filter.RLock()
	defer filter.RUnlock()
	return !filter.crawledURLs[url]
}

func (filter *Filter) UpdateCrawledURLs(url string) {
	filter.Lock()
	filter.crawledURLs[url] = true
	filter.Unlock()
}
