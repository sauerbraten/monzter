package main

import "sync"

type result struct {
	link string
	tree LinkTree
	err  error
}

func funnel(channels ...<-chan result) <-chan result {
	combined := make(chan result)

	var wg sync.WaitGroup
	wg.Add(len(channels))

	for _, ch := range channels {
		go func(ch <-chan result) {
			for res := range ch {
				combined <- res
			}
			wg.Done()
		}(ch)
	}

	go func() {
		wg.Wait()
		close(combined)
	}()

	return combined
}
