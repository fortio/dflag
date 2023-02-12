// Copyright (c) Improbable Worlds Ltd, Fortio Authors. All Rights Reserved
// See LICENSE for licensing terms.

package dflag

import (
	"flag"
	"fmt"
)

// Would use just
// type DynBoolValue = DynValue[bool]
// but that doesn't work with IsBoolFlag
// https://github.com/golang/go/issues/53473
// only fixed in go 1.20
// so we extend this type as special, the only one with that method.

func NewBool(value bool, usage string) *DynBoolValue {
	dynValue := DynBoolValue{}
	dynInit(&dynValue.DynValue, value, usage)
	return &dynValue
}

func FlagBool(name string, o *DynBoolValue) *DynBoolValue {
	return FlagSetBool(flag.CommandLine, name, o)
}

func FlagSetBool(flagSet *flag.FlagSet, name string, dynValue *DynBoolValue) *DynBoolValue {
	// we inline/repeat code of FlagSet(flagSet, name, &dynValue.DynValue)
	// because we need to set this specific type for the IsBoolFlag to work
	dynValue.flagSet = flagSet
	dynValue.flagName = name
	flagSet.Var(dynValue, name, dynValue.usage)
	flagSet.Lookup(name).DefValue = fmt.Sprintf("%v", dynValue.av.Load())
	return dynValue
}

// DynBool creates a `Flag` that represents `bool` which is safe to change dynamically at runtime.
func DynBool(flagSet *flag.FlagSet, name string, value bool, usage string) *DynBoolValue {
	return FlagSetBool(flagSet, name, NewBool(value, usage))
}

// DynStringSetValue implements a dynamic set of strings.
type DynBoolValue struct {
	DynamicBoolValueTag
	DynValue[bool]
}
