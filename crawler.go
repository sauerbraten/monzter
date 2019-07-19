package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
	"golang.org/x/time/rate"
)

// Crawler is a HTML crawler with configurable maximum depth and rate limit.
// Instances of Crawler are not safe for concurrent use.
type Crawler struct {
	visited  SchemelessURLSet // to avoid circular crawling
	client   *http.Client
	root     *url.URL
	maxDepth int
	limiter  *rate.Limiter
}

// NewCrawler returns a crawler ready to crawl the page at the specified link,
// limited to the specified maximum depth and outgoing request rate.
func NewCrawler(link string, maxDepth int, maxReqsPerSecond float64) (*Crawler, error) {
	root, err := url.Parse(link)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", link)
	}

	if !root.IsAbs() {
		return nil, errors.New("you must specify an absolute URL")
	}

	if root.Path == "" {
		root.Path = "/"
	}

	return &Crawler{
		visited: SchemelessURLSet{},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		root:     root,
		maxDepth: maxDepth,
		// the following line creates a limiter with burst = 1.
		// burst of 1 is fine; we never use limiter.*N methods.
		limiter: rate.NewLimiter(rate.Limit(maxReqsPerSecond), 1),
	}, nil
}

// Run returns link tree found on the page specified by c.root (= the link passed
// to NewCrawler). Run recursively walks the page tree, following only links with
// the same hostname as the link passed into NewCrawler, to the maximum depth and
// with the maximum outgoing request rate configured in NewCrawler.
func (c *Crawler) Run() (URLTree, error) {
	c.visited.EnsureContains(c.root)
	return c.crawl(c.root, 0)
}

// crawl returns the link tree found on the page specified by pageURL. crawl recursively
// walks the page tree, following only links with the same hostname as c.root, and only
// to the maximum depth configured in c.maxDepth. The number of outgoing requests is
// limited by c.limiter.
func (c *Crawler) crawl(pageURL *url.URL, depth int) (URLTree, error) {
	// fmt.Println("crawling", pageURL)

	// make sure we stay nice
	err := c.limiter.Wait(context.Background())
	if err != nil {
		return nil, errors.Wrapf(err, "error while waiting to be allowed to crawl %s", pageURL)
	}

	page, err := c.fetch(pageURL)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching %s", pageURL)
	}

	hrefs, err := c.uniqueLinksInPage(page, pageURL)
	page.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "error finding links on %s", pageURL)
	}

	tree := URLTree{}

	for _, href := range hrefs {
		// fmt.Println(pageURL, "links to", href)

		// store url in this page's tree, with no sub tree for now
		tree[href] = nil

		// mark this link as visited, if not already
		if c.visited.EnsureContains(href) {
			// don't crawl if it was already marked visited
			// fmt.Println("skipping visited link", href)
			continue
		}

		// don't crawl too deep, or on the wrong host
		if depth+1 >= c.maxDepth || href.Hostname() != c.root.Hostname() {
			continue
		}

		// crawl the linked page and return its sub tree
		subTree, err := c.crawl(href, depth+1)
		if err != nil {
			return nil, err
		}

		// store sub tree of links found on linked page in tree
		tree[href] = subTree
	}

	return tree, nil
}

// parseToAbsURL parses the provided string as URL and if necessary
// resolves it to an absolute URL using c.root as base. parseToAbsURL
// also ensures the returned URL has a non-empty path.
func parseToAbsURL(href string, base *url.URL) (*url.URL, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return nil, err
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return base.ResolveReference(parsed), nil
}

// fetch requests the specified page and returns the response body.
func (c *Crawler) fetch(pageURL *url.URL) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "monzter")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// uniqueLinksInPage parses the contents of page as HTML and returns all
// unique URLs found in href attributes of all <a> tags in the HTML tree.
func (c *Crawler) uniqueLinksInPage(page io.Reader, pageURL *url.URL) ([]*url.URL, error) {
	doc, err := html.Parse(page)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing page")
	}

	// walkLinks recursively walks the HTML tree and
	// calls visitLink on every <a> tag it encounters
	var walkLinks func(*html.Node, func(*html.Node))
	walkLinks = func(n *html.Node, visitLink func(n *html.Node)) {
		if n.Type == html.ElementNode && n.Data == "a" {
			visitLink(n)
			// let's assume there are no nested <a> tags
			// https://html.spec.whatwg.org/#the-a-element
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkLinks(c, visitLink)
		}
	}

	links := []*url.URL{}
	seenOnPage := SchemelessURLSet{}

	walkLinks(doc, func(n *html.Node) {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				href, err := parseToAbsURL(attr.Val, pageURL)
				if err != nil {
					// ignore malformed links
					return
				}
				if !seenOnPage.EnsureContains(href) {
					links = append(links, href)
				}
				return
			}
		}
	})

	return links, nil
}
