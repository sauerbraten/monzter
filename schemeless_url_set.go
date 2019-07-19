package main

import (
	"net/url"
)

// SchemelessURLSet uses the scheme-less equivalent of a URL u
// to determine whether u is an element of the set or not.
type SchemelessURLSet map[string]struct{}

// EnsureContains adds u and returns false in case u was not contained
// in s before. Otherwise (if u already was in s), it returns true.
func (s SchemelessURLSet) EnsureContains(u *url.URL) (existed bool) {
	key := withoutScheme(u)
	if _, ok := s[key]; ok {
		return true
	}
	s[key] = struct{}{}
	return false
}

func withoutScheme(u *url.URL) string {
	clone, _ := url.Parse(u.String())
	clone.Scheme = ""
	return clone.String()
}
