package deps

import (
	"testing"
)

func TestParseRequirements(t *testing.T) {
	content := `
# This is a comment
requests==2.31.0
numpy>=1.20,<2.0
gunicorn
-r other-requirements.txt
  pandas ~= 1.3.0  
`
	deps := ParseRequirements(content)
	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	expected := []struct {
		name    string
		version string
	}{
		{"requests", "2.31.0"},
		{"numpy", "1.20"},
		{"gunicorn", ""},
		{"pandas", "1.3.0"},
	}

	for i, exp := range expected {
		if deps[i].Name != exp.name {
			t.Errorf("expected name %s, got %s", exp.name, deps[i].Name)
		}
		if deps[i].Version != exp.version {
			t.Errorf("expected version %s, got %s", exp.version, deps[i].Version)
		}
		if deps[i].Language != "python3" {
			t.Errorf("expected language python3, got %s", deps[i].Language)
		}
	}
}

func TestParsePackageJson(t *testing.T) {
	content := `{
  "dependencies": {
    "lodash": "^4.17.21",
    "express": "~4.18.2"
  },
  "devDependencies": {
    "typescript": "^5.0.4"
  }
}`
	deps, err := ParsePackageJson(content)
	if err != nil {
		t.Fatalf("failed to parse package.json: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	expected := []struct {
		name    string
		version string
		remark  string
	}{
		{"lodash", "4.17.21", ""},
		{"express", "4.18.2", ""},
		{"typescript", "5.0.4", "devDependencies"},
	}

	for i, exp := range expected {
		if deps[i].Name != exp.name {
			t.Errorf("expected name %s, got %s", exp.name, deps[i].Name)
		}
		if deps[i].Version != exp.version {
			t.Errorf("expected version %s, got %s", exp.version, deps[i].Version)
		}
		if deps[i].Remark != exp.remark {
			t.Errorf("expected remark %s, got %s", exp.remark, deps[i].Remark)
		}
		if deps[i].Language != "node" {
			t.Errorf("expected language node, got %s", deps[i].Language)
		}
	}
}
