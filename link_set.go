package main

import "sync"

type LinkSet struct {
	// a *sync.Map is used here instead of a map[string]struct{}
	// and *sync.RWMutex, because it is optimized for this use
	// case: writing a key-value pair once, then reading it often.
	m *sync.Map
}

func NewLinkSet() *LinkSet {
	return &LinkSet{
		m: new(sync.Map),
	}
}

func (s LinkSet) EnsureContains(url string) (existed bool) {
	_, existed = s.m.LoadOrStore(url, struct{}{})
	return
}
