package deps

import (
	"testing"
)

func TestParsePythonVersionFromShow(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "standard pip show output",
			output: `Name: requests
Version: 2.31.0
Summary: Python HTTP for Humans.
Home-page: https://requests.readthedocs.io
Author: Kenneth Reitz
License: Apache 2.0
Location: /root/.local/share/mise/installs/python/3.11/lib/python3.11/site-packages
`,
			expected: "2.31.0",
		},
		{
			name: "lowercase version prefix",
			output: `name: urllib3
version: 1.26.18
location: /site-packages
`,
			expected: "1.26.18",
		},
		{
			name: "package not found warning",
			output: `WARNING: Package(s) not found: notfoundpkg
`,
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePythonVersionFromShow(tt.output)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestParseNpmListOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   []byte
		language string
		expected map[string]string
	}{
		{
			name: "standard npm list json",
			output: []byte(`{
  "dependencies": {
    "pm2": {
      "version": "5.3.0",
      "overridden": false
    },
    "typescript": {
      "version": "5.4.2"
    }
  }
}`),
			language: "node",
			expected: map[string]string{
				"pm2":        "5.3.0",
				"typescript": "5.4.2",
			},
		},
		{
			name:     "empty json",
			output:   []byte(`{}`),
			language: "node",
			expected: map[string]string{},
		},
		{
			name:     "invalid json",
			output:   []byte(`npm ERR! missing`),
			language: "node",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgs := parseNpmListOutput(tt.output, tt.language)
			if tt.expected == nil {
				if pkgs != nil {
					t.Errorf("expected nil, got %v", pkgs)
				}
				return
			}
			if len(pkgs) != len(tt.expected) {
				t.Fatalf("expected %d packages, got %d", len(tt.expected), len(pkgs))
			}
			for _, p := range pkgs {
				expVer, ok := tt.expected[p.Name]
				if !ok {
					t.Errorf("unexpected package %s", p.Name)
					continue
				}
				if p.Version != expVer {
					t.Errorf("expected %s version %s, got %s", p.Name, expVer, p.Version)
				}
				if p.Language != tt.language {
					t.Errorf("expected language %s, got %s", tt.language, p.Language)
				}
			}
		})
	}
}

