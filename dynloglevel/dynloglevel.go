// Copyright 2023 Fortio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Sets up a dynamic flag to change the log level dynamically at runtime.
// Use in conjunction with endpoint or configmap.
// Code was initially in fortio.org/fortio/log before log was extracted to its own repo
// without any dependencies.
package dynloglevel

import (
	"flag"
	"fmt"
	"strings"

	"fortio.org/dflag"
	"fortio.org/log"
)

var done = false

// LoggerFlagSetup sets up the `loglevel` flag as a dynamic flag
// (or another name if desired/passed).
func LoggerFlagSetup(optionalFlagName ...string) {
	if done {
		return // avoid redefining flag/make it ok for multiple function to init this.
	}
	// virtual dynLevel flag that maps back to actual level
	defVal := log.GetLogLevel().String()
	usage := fmt.Sprintf("log `level`, one of %v", log.LevelToStrA)
	flag := dflag.New(defVal, usage).WithInputMutator(
		func(inp string) string {
			// The validation map has full lowercase and capitalized first letter version
			return strings.ToLower(strings.TrimSpace(inp))
		}).WithValidator(
		func(newStr string) error {
			_, err := log.ValidateLevel(newStr)
			return err
		}).WithSyncNotifier(
		func(old, newStr string) {
			_ = log.SetLogLevelStr(newStr) // will succeed as we just validated it first
		})
	if len(optionalFlagName) == 0 {
		optionalFlagName = []string{"loglevel"}
	}
	for _, name := range optionalFlagName {
		dflag.Flag(name, flag)
	}
	done = true
}

// ChangeFlagsDefault sets some flags to a different default.
// Will panic/exist if the flag is not found.
func ChangeFlagsDefault(newDefault string, flagNames ...string) {
	for _, flagName := range flagNames {
		f := flag.Lookup(flagName)
		if f == nil {
			log.Fatalf("flag %s not found", flagName) //nolint:revive // we know it's unreachable after
			continue                                  // not reached but linter doesn't know Fatalf panics/exits
		}
		f.DefValue = newDefault
		err := f.Value.Set(newDefault)
		if err != nil {
			log.Fatalf("error setting flag %s: %v", flagName, err)
		}
	}
}
