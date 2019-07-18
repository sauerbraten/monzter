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

type Crawler struct {
	visited  *LinkSet // to avoid circular crawling
	client   *http.Client
	root     *url.URL
	maxDepth int
	limiter  *rate.Limiter
}

func NewCrawler(link string, maxDepth int, maxReqsPerSecond float64) (*Crawler, error) {
	root, err := url.Parse(link)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", link)
	}

	if !root.IsAbs() {
		return nil, errors.New("you must specify an absolute URL")
	}

	return &Crawler{
		visited: NewLinkSet(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		root:     root,
		maxDepth: maxDepth,
		// the following line creates a limiter with burst = 1
		// burst of 1 is fine; we never use limiter.*N methods
		limiter: rate.NewLimiter(rate.Limit(maxReqsPerSecond), 1),
	}, nil
}

func (c *Crawler) Run() (LinkTree, error) {
	url, err := c.parseToAbsURL(c.root.String())
	if err != nil {
		// should never happen
		return nil, errors.Wrapf(err, "error normalizing link %s", c.root.String())
	}

	// use the absolute URL
	return c.crawl(url, 0)
}

func (c *Crawler) crawl(u *url.URL, depth int) (LinkTree, error) {
	// mark this link as visited, if not already
	if c.visited.EnsureContains(urlWithoutScheme(u)) {
		// don't crawl if it was already marked visited
		return nil, nil
	}

	// don't crawl too deep, or on the wrong host
	if depth == c.maxDepth || u.Hostname() != c.root.Hostname() {
		return nil, nil
	}

	// make sure we are a Nice Crawler
	err := c.limiter.Wait(context.Background())
	if err != nil {
		return nil, errors.Wrapf(err, "error while waiting to be allowed to crawl %s", u.String())
	}

	page, err := c.fetchPage(u.String())
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching %s", u)
	}

	links, err := c.findLinksInPage(page)
	page.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "error finding links on %s", u)
	}

	channels := make([]<-chan result, len(links))

	for index, link := range links {
		url, err := c.parseToAbsURL(link)
		if err != nil {
			return nil, errors.Wrapf(err, "error normalizing link %s", link)
		}

		ch := make(chan result)
		channels[index] = ch

		go func(link string) {
			defer close(ch)
			tree, err := c.crawl(url, depth+1)
			ch <- result{
				link: url.String(),
				tree: tree,
				err:  err,
			}
		}(link)
	}

	return c.collectResults(channels...)
}

// parseToAbsURL parses the provided string as URL and if necessary
// resolves it to an absolute URL using c.root as base.
func (c *Crawler) parseToAbsURL(link string) (*url.URL, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return c.root.ResolveReference(parsed), nil
}

func (c *Crawler) fetchPage(link string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
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

func (c *Crawler) findLinksInPage(page io.Reader) (links []string, err error) {
	doc, err := html.Parse(page)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing page")
	}

	seenOnPage := NewLinkSet()

	walkLinks(doc, func(n *html.Node) {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				href, err := url.Parse(attr.Val)
				if err != nil {
					// ignore malformed links
					return
				}
				if !seenOnPage.EnsureContains(urlWithoutScheme(href)) {
					links = append(links, attr.Val)
					return
				}
			}
		}
	})

	return
}

func (c *Crawler) collectResults(channels ...<-chan result) (LinkTree, error) {
	tree := LinkTree{}

	combined := funnel(channels...)

	for res := range combined {
		if res.err != nil {
			return nil, res.err
		}

		tree[res.link] = res.tree
	}

	return tree, nil
}

func urlWithoutScheme(u *url.URL) string {
	clone, _ := url.Parse(u.String())
	clone.Scheme = ""
	return clone.String()
}

func walkLinks(n *html.Node, visitLink func(n *html.Node)) {
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
