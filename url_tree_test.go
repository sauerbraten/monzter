package main

import (
	"net/url"
	"testing"
)

var (
	foo *url.URL
	bar *url.URL
	baz *url.URL
)

func init() {
	foo, _ = url.Parse("http://example.com/foo")
	bar, _ = url.Parse("http://example.com/bar")
	baz, _ = url.Parse("http://example.com/baz")
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		input    URLTree
		expected string
	}{
		{
			name: "foo bar baz",
			input: URLTree{
				foo: {
					bar: nil,
					baz: nil,
				},
				bar: {
					baz: nil,
				},
				baz: nil,
			},
			expected: `http://example.com/bar
  http://example.com/baz
http://example.com/baz
http://example.com/foo
  http://example.com/bar
  http://example.com/baz
`,
		},
		{
			name: "single element",
			input: URLTree{
				foo: nil,
			},
			expected: `http://example.com/foo
`,
		},
		{
			name:     "empty tree",
			input:    URLTree{},
			expected: ``,
		},
		{
			name: "deep nesting",
			input: URLTree{
				foo: {
					foo: {
						foo: {
							foo: {
								foo: {
									foo: {
										foo: nil,
									},
								},
							},
						},
					},
				},
			},
			expected: `http://example.com/foo
  http://example.com/foo
    http://example.com/foo
      http://example.com/foo
        http://example.com/foo
          http://example.com/foo
            http://example.com/foo
`,
		},
	}

	for _, test := range tests {
		output := test.input.String()
		if output != test.expected {
			t.Errorf("test '%s' failed: expected %s, but got %s", test.name, test.expected, output)
		}
	}
}
