package main

import (
	"net/url"
)

type SchemelessURLSet map[string]struct{}

// EnsureContains adds u and returns false in case u was not contained
// in s before. Otherwise (if u already was in s), it returns true.
// EnsureContains ignores u's scheme when determinining whether u is
// already contained in s or not.
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
