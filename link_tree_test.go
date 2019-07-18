package main

import "testing"

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		input    LinkTree
		expected string
	}{
		{
			name: "foo bar baz",
			input: LinkTree{
				"foo": {
					"bar": nil,
					"baz": nil,
				},
				"bar": {
					"baz": nil,
				},
				"baz": nil,
			},
			expected: `- bar:
  - baz
- baz
- foo:
  - bar
  - baz
`,
		},
		{
			name: "single element",
			input: LinkTree{
				"foo": nil,
			},
			expected: `- foo
`,
		},
		{
			name:     "deep nesting",
			input:    LinkTree{},
			expected: ``,
		},
		{
			input: LinkTree{
				"foo": {
					"foo": {
						"foo": {
							"foo": {
								"foo": {
									"foo": {
										"foo": nil,
									},
								},
							},
						},
					},
				},
			},
			expected: `- foo:
  - foo:
    - foo:
      - foo:
        - foo:
          - foo:
            - foo
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
