package main

import (
	"flag"
	"fmt"
	"os"
)

type configuration struct {
	maxDepth          int
	requestsPerSecond float64
	entrypoint        string
}

func parseArgs() *configuration {
	c := &configuration{}

	flag.IntVar(&c.maxDepth, "depth", 1, "maximum depth to crawl links")
	flag.Float64Var(&c.requestsPerSecond, "rate", 10.0, "maximum number of requests per second")

	flag.Parse()

	c.entrypoint = flag.Arg(0)
	if c.entrypoint == "" {
		fmt.Fprintln(os.Stderr, "no entrypoint provided!")
		flag.Usage()
		os.Exit(2)
	}

	return c
}

func main() {
	config := parseArgs()

	c, err := NewCrawler(config.entrypoint, config.maxDepth, config.requestsPerSecond)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	tree, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Print(tree)
}
