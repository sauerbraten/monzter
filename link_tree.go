package main

import (
	"bytes"
	"sort"
	"strings"
)

type LinkTree map[string]LinkTree

func (t LinkTree) String() string {
	return t.string(0)
}

func (t LinkTree) string(indent int) string {
	b := strings.Builder{}

	// sort pages
	pages := make([]string, 0, len(t))
	for page := range t {
		pages = append(pages, page)
	}

	sort.Strings(pages)

	for _, page := range pages {
		b.Write(bytes.Repeat([]byte("  "), indent))
		b.WriteString("- ")
		b.WriteString(page)
		if len(t[page]) != 0 {
			b.WriteString(":\n")
			b.WriteString(t[page].string(indent + 1))
		} else {
			b.WriteString("\n")
		}
	}

	return b.String()
}
