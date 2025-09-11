// Copyright 2015 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package dflag

import (
	"errors"
	"flag"
	"testing"
	"time"

	"fortio.org/assert"
	"fortio.org/duration"
)

func TestDynDuration_SetAndGet(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	dynFlag := DynDuration(set, "some_duration_1", 5*time.Second, "Use it or lose it")
	assert.Equal(t, 5*time.Second, dynFlag.Get(), "value must be default after create")
	err := set.Set("some_duration_1", "1d3h\n")
	assert.NoError(t, err, "setting value must succeed")
	assert.Equal(t, 27*time.Hour, dynFlag.Get(), "value must be set after update")
	// new string version even for regular duration:
	assert.Equal(t, "1d3h", dynFlag.String(), "string representation must match")
	err = set.Set("some_duration_1", "not-a-duration")
	assert.Error(t, err, "setting bogus value should fail")
}

func TestDynDuration_IsMarkedDynamic(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	DynDuration(set, "some_duration_1", 5*time.Minute, "Use it or lose it")
	assert.True(t, IsFlagDynamic(set.Lookup("some_duration_1")))
}

func TestDynDuration_FiresValidators(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	validator := func(x time.Duration) error {
		if x > 1*time.Hour {
			return errors.New("too long")
		}
		return nil
	}
	DynDuration(set, "some_duration_1", 5*time.Second, "Use it or lose it").WithValidator(validator)

	assert.NoError(t, set.Set("some_duration_1", "50m"), "no error from validator when in range")
	assert.Error(t, set.Set("some_duration_1", "2h"), "error from validator when value out of range")
}

func TestDynDuration_FiresNotifier(t *testing.T) {
	waitCh := make(chan bool, 1)
	notifier := func(oldVal time.Duration, newVal time.Duration) {
		assert.EqualValues(t, 5*time.Second, oldVal, "old value in notify must match previous value")
		assert.EqualValues(t, 30*time.Second, newVal, "new value in notify must match set value")
		waitCh <- true
	}

	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	DynDuration(set, "some_duration_1", 5*time.Second, "Use it or lose it").WithNotifier(notifier)
	set.Set("some_duration_1", "30s")
	select {
	case <-time.After(notifierTimeout):
		assert.Fail(t, "failed to trigger notifier")
	case <-waitCh:
	}
}

func TestDynDurationDuration(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	d := Dyn(set, "some_duration_1", duration.Duration(1*duration.Week), "Test of week")
	assert.Equal(t, 1*duration.Week, time.Duration(d.Get()), "default value must be correct")
	err := d.Set("3w2d4h")
	assert.NoError(t, err, "setting valid duration must succeed")
	assert.Equal(t, 3*duration.Week+2*duration.Day+4*time.Hour, time.Duration(d.Get()), "set value must be correct")
	// string version
	assert.Equal(t, "3w2d4h", d.String(), "string representation must match")
}

func Benchmark_Duration_Dyn_Get(b *testing.B) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	value := DynDuration(set, "some_duration_1", 5*time.Second, "Use it or lose it")
	set.Set("some_duration_1", "10s")
	for i := 0; i < b.N; i++ {
		value.Get().Nanoseconds()
	}
}

func Benchmark_Duration_Normal_get(b *testing.B) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	valPtr := set.Duration("some_duration_1", 5*time.Second, "Use it or lose it")
	set.Set("some_duration_1", "10s")
	for i := 0; i < b.N; i++ {
		valPtr.Nanoseconds()
	}
}
