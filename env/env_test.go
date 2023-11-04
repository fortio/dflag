package env_test

import (
	"os"
	"strings"
	"testing"

	"fortio.org/assert"
	"fortio.org/dflag/env"
	"fortio.org/log"
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

type Embedded struct {
	InnerA string
	InnerB string
}

type HiddenEmbedded struct {
	HA string
	HB string
}

type FooConfig struct {
	Foo          string
	Bar          string
	Blah         int `env:"A_SPECIAL_BLAH"`
	ABool        bool
	NotThere     int `env:"-"`
	HTTPServer   string
	IntPointer   *int
	FloatPointer *float64
	WontShowYet  map[string]string
	Embedded
	HiddenEmbedded `env:"-"`
	RecurseHere    Embedded
}

func TestStructToEnvVars(t *testing.T) {
	log.SetLogLevelQuiet(log.Verbose)
	intV := 199
	foo := FooConfig{
		Foo:          "a\nfoo with \" quotes and \\ and '",
		Bar:          "42str",
		Blah:         42,
		ABool:        true,
		NotThere:     13,
		HTTPServer:   "http://localhost:8080",
		IntPointer:   &intV,
		FloatPointer: nil,
		RecurseHere: Embedded{
			InnerA: "rec a",
			InnerB: "rec b",
		},
	}
	foo.InnerA = "inner a"
	foo.InnerB = "inner b"
	empty := env.StructToEnvVars(42) // error/empty
	if len(empty) != 0 {
		t.Errorf("expected empty, got %v", empty)
	}
	envVars := env.StructToEnvVars(&foo)
	if len(envVars) != 11 {
		t.Errorf("expected 11 env vars, got %d: %+v", len(envVars), envVars)
	}
	str := env.ToShellWithPrefix("TST_", envVars)
	//nolint:lll
	expected := `TST_FOO="a\nfoo with \" quotes and \\ and '"
TST_BAR="42str"
TST_A_SPECIAL_BLAH="42"
TST_A_BOOL=true
TST_HTTP_SERVER="http://localhost:8080"
TST_INT_POINTER="199"
TST_FLOAT_POINTER=
TST_INNER_A="inner a"
TST_INNER_B="inner b"
TST_RECURSE_HERE_INNER_A="rec a"
TST_RECURSE_HERE_INNER_B="rec b"
export TST_FOO TST_BAR TST_A_SPECIAL_BLAH TST_A_BOOL TST_HTTP_SERVER TST_INT_POINTER TST_FLOAT_POINTER TST_INNER_A TST_INNER_B TST_RECURSE_HERE_INNER_A TST_RECURSE_HERE_INNER_B
`
	if str != expected {
		t.Errorf("\n---expected:---\n%s\n---got:---\n%s", expected, str)
	}
}

func TestSetFromEnv(t *testing.T) {
	log.SetLogLevelQuiet(log.Verbose)
	foo := FooConfig{}
	envs := []struct {
		k string
		v string
	}{
		{"TST2_FOO", "another\nfoo"},
		{"TST2_BAR", "bar"},
		{"TST2_RECURSE_HERE_INNER_B", "in1"},
		{"TST2_A_SPECIAL_BLAH", "31"},
		{"TST2_A_BOOL", "1"},
		{"TST2_FLOAT_POINTER", "5.75"},
		{"TST2_INT_POINTER", "73"},
	}
	for _, e := range envs {
		os.Setenv(e.k, e.v)
	}
	env.SetFromEnv("TST2_", &foo)
	assert.Equal(t, foo.Foo, "another\nfoo")
	assert.Equal(t, foo.Bar, "bar")
	assert.Equal(t, foo.RecurseHere.InnerB, "in1")
	assert.Equal(t, foo.Blah, 31)
	assert.Equal(t, foo.ABool, true)
	assert.NotEqual(t, foo.FloatPointer, nil)
	assert.Equal(t, *foo.FloatPointer, 5.75)
	assert.NotEqual(t, foo.IntPointer, nil)
	assert.Equal(t, *foo.IntPointer, 73)
}
