// Copyright (c) Fortio Authors, All Rights Reserved
// See LICENSE for licensing terms. (Apache-2.0)

package dflag

import (
	"flag"
	"testing"

	"fortio.org/assert"
)

// Additional generic tests, most tests are covered by the old per type tests.

func TestParse_BadType(t *testing.T) {
	_, err := parse[uint8]("23")
	assert.Error(t, err, "Expecting unpected type error")
	assert.Equal(t, err.Error(), "unexpected type uint8", "message/error should match")
}

func TestParse_GoodType(t *testing.T) {
	v, err := Parse[int64]("23")
	assert.NoError(t, err, "Shouldn't error for supported types")
	assert.Equal(t, int64(23), v)
}

func TestDflag_NonDynamic(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	set.Bool("notdyn", false, "...")
	static := set.Lookup("notdyn")
	assert.True(t, static != nil)
	assert.False(t, IsFlagDynamic(static))
}

func TestSetToString(t *testing.T) {
	s := Set[string]{"z": {}, "a": {}, "c": {}, "b": {}}
	f := New(s, "test set")
	assert.Equal(t, "a,b,c,z", s.String())
	assert.Equal(t, "a,b,c,z", f.Get().String())
}

func TestArrayToString(t *testing.T) {
	s := []string{"z", "a", "c", "b"}
	f := New(s, "test array")
	Flag("testing123", f)
	defValue := flag.CommandLine.Lookup("testing123").DefValue
	// order preserved unlike for Set where we sort
	print := f.String()
	assert.Equal(t, "z,a,c,b", print)
	assert.Equal(t, "z,a,c,b", defValue)
}
