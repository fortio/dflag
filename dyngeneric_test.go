// Copyright (c) Fortio Authors, All Rights Reserved
// See LICENSE for licensing terms. (Apache-2.0)

package dflag

import (
	"flag"
	"testing"

	"fortio.org/assert"
	"fortio.org/sets"
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
	s := sets.Set[string]{"z": {}, "a": {}, "c": {}, "b": {}}
	f := New(s, "test set")
	assert.Equal(t, "a,b,c,z", s.String())
	assert.Equal(t, "a,b,c,z", f.Get().String())
}

func TestArrayToString(t *testing.T) {
	s := []string{"z", "a", "c", "b"}
	f := New(s, "test array")
	Flag("testing123", f)
	flag := flag.CommandLine.Lookup("testing123")
	defValue := flag.DefValue
	// order preserved unlike for sets.Set where we sort
	str := f.String()
	assert.Equal(t, "z,a,c,b", str)
	assert.Equal(t, "z,a,c,b", defValue)
	b := IsBinary(flag)
	if b != nil {
		t.Errorf("flag %v isn't binary yet got non nil: %v", flag, b)
	}
}

func TestRemoveCommon(t *testing.T) {
	setA := sets.New("a", "b", "c", "d")
	setB := sets.New("b", "d", "e", "f", "g")
	setAA := setA.Clone()
	setBB := setB.Clone()
	sets.RemoveCommon(setAA, setBB)
	assert.Equal(t, "a,c", setAA.String())   // removed
	assert.Equal(t, "e,f,g", setBB.String()) // added
	// Swap order to exercise the optimization on length of iteration
	// also check clone is not modifying the original etc
	setAA = setB.Clone() // putting B in AA on purpose and vice versa
	setBB = setA.Clone()
	sets.RemoveCommon(setAA, setBB)
	assert.Equal(t, "a,c", setBB.String())
	assert.Equal(t, "e,f,g", setAA.String())
	assert.True(t, setBB.Has("c"))
	setBB.Remove("c")
	assert.False(t, setBB.Has("c"))
}

func TestBinary(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	dynFlag := Dyn(set, "some_binary", []byte{2, 1, 0}, "some binary values")
	assert.Equal(t, []byte{2, 1, 0}, dynFlag.Get(), "value must be default after create")
	err := set.Set("some_binary", "\nAAEC\n") // extra newlines are fine
	assert.NoError(t, err, "setting value must succeed")
	assert.Equal(t, []byte{0, 1, 2}, dynFlag.Get(), "value must be set after update")
	str := dynFlag.String()
	assert.Equal(t, "AAEC", str, "value when printed must be base64 encoded")
	err = set.Set("some_binary", "foo bar")
	assert.Error(t, err, "setting bogus base64 should fail")
	flag := set.Lookup("some_binary")
	assert.True(t, IsFlagDynamic(flag), "flag must be dynamic")
	if IsBinary(flag) == nil {
		t.Errorf("flag %v isn't binary yet it should", flag)
	}
}
