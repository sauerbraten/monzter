package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// setupServer starts a httptest.Server and uses setupRoutes to configure routing.
// setupServer returns the servers base URL as scheme and host, as well as a function
// to stop the server after testing finished.
func setupServer(setupRoutes func(setupRoute func(pattern string, links []string))) (scheme, host string, tearDown func()) {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)

	u, _ := url.Parse(ts.URL)

	setupRoutes(func(pattern string, links []string) {
		page := `<!doctype html><html lang="en"><head><title>No Links</title></head><body>`
		for _, link := range links {
			link = replacePlaceholders(link, u.Scheme, u.Host)
			page += fmt.Sprintf(`<a href="%s">%s</a>`, link, link)
		}
		page += `<p>This is a non-anchor node.</p></body></html>`

		mux.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(page))
		})
	})

	return u.Scheme, u.Host, ts.Close
}

func replacePlaceholders(s, scheme, host string) string {
	s = strings.ReplaceAll(s, "{scheme}", scheme)
	s = strings.ReplaceAll(s, "{host}", host)
	s = strings.ReplaceAll(s, "{base}", scheme+"://"+host)
	return s
}

func TestCrawler(t *testing.T) {
	// a test is defined as a struct
	// the links argument passed to setupRoute() and the expected output
	// can contain '{scheme}', '{host}' and '{base}' (= '{scheme}://{host}')
	// as placeholders for the scheme, host, and base URL of the test server
	tests := []struct {
		name             string
		setupRoutes      func(setupRoute func(pattern string, links []string))
		maxDepth         int
		maxReqsPerSecond float64
		expected         string
	}{
		{
			name: "absolute links",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"{base}/foo"})
				setupRoute("/foo", []string{"{base}/baz"})
				setupRoute("/baz", nil)
			},
			maxDepth:         5,
			maxReqsPerSecond: 100.0,
			expected: `{base}/foo
  {base}/baz
`,
		},
		{
			name: "relative links",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"/foo/bar"})
				setupRoute("/foo/bar", []string{"baz"})
				setupRoute("/foo/baz", []string{"../../baz"})
				setupRoute("/baz", []string{"/foo", "foo/bar"})
			},
			maxDepth:         5,
			maxReqsPerSecond: 100.0,
			expected: `{base}/foo/bar
  {base}/foo/baz
    {base}/baz
      {base}/foo
      {base}/foo/bar
`,
		},
		{
			name: "equivalent relative and absolute links are only listed once",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"/foo", "{base}/foo"})
				setupRoute("/foo", nil)
			},
			maxDepth:         5,
			maxReqsPerSecond: 100.0,
			expected: `{base}/foo
`,
		},
		{
			name: "circular links are detected",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"/depth1", "/depth4"})
				setupRoute("/depth1", []string{"/depth2"})
				setupRoute("/depth2", []string{"/depth3"})
				setupRoute("/depth3", []string{"/depth4"})
				setupRoute("/depth4", []string{"/entry"})
			},
			maxDepth:         100,
			maxReqsPerSecond: 100.0,
			expected: `{base}/depth1
  {base}/depth2
    {base}/depth3
      {base}/depth4
        {base}/entry
{base}/depth4
`,
		},
		{
			// same routes as above, but maximum depth is now 2
			// this should result in flatter output
			name: "limiting depth works",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"/depth1", "/depth4"})
				setupRoute("/depth1", []string{"/depth2"})
				setupRoute("/depth2", []string{"/depth3"})
				setupRoute("/depth3", []string{"/depth4"})
				setupRoute("/depth4", []string{"/entry"})
			},
			maxDepth:         2,
			maxReqsPerSecond: 100.0,
			expected: `{base}/depth1
  {base}/depth2
{base}/depth4
  {base}/entry
`,
		},
		{
			name: "external links",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"facebook.com", "http://github.com", "https://google.com/maps", "//www.wikipedia.org"})
				setupRoute("/facebook.com", nil) // <a href="facebook.com"> is a local link!
			},
			maxDepth:         5,
			maxReqsPerSecond: 100.0,
			expected: `{base}/facebook.com
http://github.com/
https://google.com/maps
{scheme}://www.wikipedia.org/
`,
		},
		{
			name: "//foo.com links get scheme of entrypoint",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"//facebook.com", "//{host}/depth1"})
				setupRoute("/depth1", []string{"//{host}/depth2"})
				setupRoute("/depth2", []string{"//{host}/depth3"})
				setupRoute("/depth3", []string{"//{host}/depth4"})
				setupRoute("/depth4", []string{"//{host}/depth5"})
				setupRoute("/depth5", []string{"//{host}/depth6"})
			},
			maxDepth:         5,
			maxReqsPerSecond: 100.0,
			expected: `{base}/depth1
  {base}/depth2
    {base}/depth3
      {base}/depth4
        {base}/depth5
{scheme}://facebook.com/
`,
		},
		{
			name: "ignoring scheme",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"http://google.com/mail", "https://google.com/mail"})
			},
			maxDepth:         100,
			maxReqsPerSecond: 100.0,
			expected: `{scheme}://google.com/mail
`,
		},
		{
			name: "links ending with slash are distinct",
			setupRoutes: func(setupRoute func(pattern string, links []string)) {
				setupRoute("/entry", []string{"http://google.com/mail", "http://google.com/mail/", "/bla", "/bla/"})
				setupRoute("/bla", nil)
				setupRoute("/bla/", nil)
			},
			maxDepth:         100,
			maxReqsPerSecond: 100.0,
			expected: `{base}/bla
{base}/bla/
{scheme}://google.com/mail
{scheme}://google.com/mail/
`,
		},
	}

	for _, test := range tests {
		scheme, host, tearDown := setupServer(test.setupRoutes)
		entrypoint := replacePlaceholders("{base}/entry", scheme, host)

		c, err := NewCrawler(entrypoint, test.maxDepth, test.maxReqsPerSecond)
		if err != nil {
			t.Error(err)
		}

		tree, err := c.Run()
		if err != nil {
			t.Error(err)
		}

		output := tree.String()
		expected := replacePlaceholders(test.expected, scheme, host)

		if output != expected {
			t.Errorf("test '%s' failed: expected\n%sbut got\n%s", test.name, expected, output)
		}

		tearDown()
	}
}
