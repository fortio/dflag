package env_test

import (
	"strings"
	"testing"

	"fortio.org/assert"
	"fortio.org/dflag/env"
)

func TestSplitByCase(t *testing.T) {
	tests := []struct {
		in  string
		out []string
	}{
		{"", nil},
		{"http2Server", []string{"http2", "Server"}},
		{"HTTPSServer42", []string{"HTTPS", "Server42"}},
		{"1", []string{"1"}},
		{"1a", []string{"1a"}},
		{"1a2Bb", []string{"1a2", "Bb"}}, // note 1a2B doesn't split
		{"a", []string{"a"}},
		{"A", []string{"A"}},
		{"Ab", []string{"Ab"}},
		{"AB", []string{"AB"}},
		{"AB", []string{"AB"}},
		{"ABC", []string{"ABC"}},
		{"ABCd", []string{"AB", "Cd"}},
		{"aa", []string{"aa"}},
		{"aaA", []string{"aa", "A"}},
		{"AAb", []string{"A", "Ab"}},
		{"aaBbbCcc", []string{"aa", "Bbb", "Ccc"}},
		{"AaBbbCcc", []string{"Aa", "Bbb", "Ccc"}},
		{"AABbbCcc", []string{"AA", "Bbb", "Ccc"}},
	}
	for _, test := range tests {
		got := env.SplitByCase(test.in)
		assert.Equal(t, got, test.out, "mismatch for", test.in)
	}
}

// TestCamelCaseToSnakeCase tests the CamelCaseToUpperSnakeCase and CamelCaseToLowerSnakeCase functions.
func TestCamelCaseToSnakeCase(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"a", "A"},
		{"A", "A"},
		{"Ab", "AB"},
		{"AB", "AB"},
		{"ABCd", "AB_CD"},
		{"aa", "AA"},
		{"aaA", "AA_A"},
		{"AAb", "A_AB"},
		{"aaBbbCcc", "AA_BBB_CCC"},
		{"http2Server", "HTTP2_SERVER"},
		{"HTTPSServer42", "HTTPS_SERVER42"},
	}
	for _, test := range tests {
		if got := env.CamelCaseToUpperSnakeCase(test.in); got != test.out {
			t.Errorf("for %q expected upper %q and got %q", test.in, test.out, got)
		}
		lower := strings.ToLower(test.out)
		if got := env.CamelCaseToLowerSnakeCase(test.in); got != lower {
			t.Errorf("for %q expected lower %q and got %q", test.in, lower, got)
		}
	}
}

func TestCamelCaseToLowerKebabCase(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"Ab", "ab"},
		{"AB", "ab"},
		{"ABCd", "ab-cd"},
		{"aa", "aa"},
		{"aaA", "aa-a"},
		{"AAb", "a-ab"},
		{"aaBbbCcc", "aa-bbb-ccc"},
		{"http2Server", "http2-server"},
		{"HTTPSServer42", "https-server42"},
	}
	for _, test := range tests {
		if got := env.CamelCaseToLowerKebabCase(test.in); got != test.out {
			t.Errorf("for %q expected %q and got %q", test.in, test.out, got)
		}
	}
}

type FooConfig struct {
	Foo        string
	Bar        string
	Blah       int `env:"A_SPECIAL_BLAH"`
	NotThere   int `env:"-"`
	HTTPServer string
}

func TestStructToEnvVars(t *testing.T) {
	foo := FooConfig{
		Foo:        "a\nfoo with \" quotes and \\ and '",
		Bar:        "42str",
		Blah:       42,
		NotThere:   13,
		HTTPServer: "http://localhost:8080",
	}
	empty := env.StructToEnvVars(42) // error/empty
	if len(empty) != 0 {
		t.Errorf("expected empty, got %v", empty)
	}
	envVars := env.StructToEnvVars(&foo)
	if len(envVars) != 4 {
		t.Errorf("expected 4 env vars, got %+v", envVars)
	}
	str := env.ToShell(envVars)
	expected := `FOO="a\nfoo with \" quotes and \\ and '"
BAR="42str"
A_SPECIAL_BLAH="42"
HTTP_SERVER="http://localhost:8080"
export FOO BAR A_SPECIAL_BLAH HTTP_SERVER
`
	if str != expected {
		t.Errorf("\n---expected:---\n%s\n---got:---\n%s", expected, str)
	}
}
