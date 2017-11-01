package main

import (
	"os"
)

func main() {
	InitCrawler(os.Args[1]).StartCrawling()
}
