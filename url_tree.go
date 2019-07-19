package main

import (
	"bytes"
	"net/url"
	"sort"
	"strings"
)

// URLTree is a tree of *url.URLs with pretty printing.
type URLTree map[*url.URL]URLTree

// String prints the tree of URLs as nested lists of URLs.
// Each URL u is printed on its own line, with indentation 2*d spaces, where d is u's
// distance from the tree root (i.e., t's entries will not be indented at all). When
// u maps to a non-nil sub tree, u is followed by the sub tree in the output.
func (t URLTree) String() string {
	return t.string(0)
}

func (t URLTree) string(indent int) string {
	b := strings.Builder{}

	// sort links, ignoring the URL scheme
	links := make(IgnoringScheme, 0, len(t))
	for link := range t {
		links = append(links, link)
	}
	sort.Sort(links)

	for _, link := range links {
		b.Write(bytes.Repeat([]byte("  "), indent))
		b.WriteString(link.String())
		b.WriteString("\n")
		if len(t[link]) != 0 {
			b.WriteString(t[link].string(indent + 1))
		}
	}

	return b.String()
}

// IgnoringScheme implements sort.Interface and sorts URLs
// by lexicographical order of their scheme-less equivalents.
type IgnoringScheme []*url.URL

func (urls IgnoringScheme) Len() int      { return len(urls) }
func (urls IgnoringScheme) Swap(i, j int) { urls[i], urls[j] = urls[j], urls[i] }
func (urls IgnoringScheme) Less(i, j int) bool {
	return withoutScheme(urls[i]) < withoutScheme(urls[j])
}
