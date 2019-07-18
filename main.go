package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/time/rate"
)

type configuration struct {
	maxDepth          int
	verbose           bool
	requestsPerSecond float64
	entrypoint        string
}

func parseArgs() *configuration {
	c := &configuration{}

	flag.IntVar(&c.maxDepth, "depth", 1, "maximum depth to crawl links")
	flag.BoolVar(&c.verbose, "v", false, "enable verbose output")
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

	// the following line creates a limiter with burst = 1
	// burst of 1 is fine; we never use limiter.*N methods
	limiter := rate.NewLimiter(rate.Limit(config.requestsPerSecond), 1)

	c, err := NewCrawler(config.entrypoint, config.maxDepth, limiter)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	tree, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Printf("Links found on %s:\n", config.entrypoint)
	fmt.Print(tree)
}
