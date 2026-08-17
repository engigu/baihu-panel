package deps

import (
	"testing"
)

func TestDetectMissingDependencies(t *testing.T) {
	tests := []struct {
		name       string
		language   string
		logContent string
		wantPkgs   []string
		wantFound  bool
	}{
		{
			name:       "Python ModuleNotFoundError",
			language:   "python3",
			logContent: "Traceback (most recent call last):\n  File \"main.py\", line 1, in <module>\n    import requests\nModuleNotFoundError: No module named 'requests'",
			wantPkgs:   []string{"requests"},
			wantFound:  true,
		},
		{
			name:       "Python No module named",
			language:   "python",
			logContent: "ImportError: No module named yaml",
			wantPkgs:   []string{"yaml"},
			wantFound:  true,
		},
		{
			name:       "Node Error Cannot find module",
			language:   "node",
			logContent: "Error: Cannot find module 'axios'\nRequire stack:\n- /app/index.js",
			wantPkgs:   []string{"axios"},
			wantFound:  true,
		},
		{
			name:       "No match",
			language:   "python",
			logContent: "Success running script",
			wantPkgs:   nil,
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkgs, gotFound := DetectMissingDependencies(tt.language, tt.logContent)
			if gotFound != tt.wantFound {
				t.Errorf("DetectMissingDependencies() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
			if len(gotPkgs) != len(tt.wantPkgs) {
				t.Errorf("DetectMissingDependencies() gotPkgs = %v, want %v", gotPkgs, tt.wantPkgs)
				return
			}
			for i, p := range gotPkgs {
				if p != tt.wantPkgs[i] {
					t.Errorf("DetectMissingDependencies() gotPkgs[%d] = %v, want %v", i, p, tt.wantPkgs[i])
				}
			}
		})
	}
}
