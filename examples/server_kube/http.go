// Copyright 2016 Michal Witkowski. All Rights Reserved.
// Copyright 2020-2023 Fortio Authors. All Rights Reserved.
// See LICENSE for licensing terms.

package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"

	"fortio.org/dflag"
	"fortio.org/dflag/configmap"
	"fortio.org/dflag/dynloglevel"
	"fortio.org/dflag/endpoint"
	"fortio.org/log"
	"fortio.org/sets"
)

//nolint:unused // some flags here unused / just for examples
var (
	listenPort = flag.Int("port", 8080, "Port the example server listens on.")
	listenHost = flag.String("host", "0.0.0.0", "Host to bind the example server to.")
	hasSetFlag = flag.Bool("has_set", true, "Whether the /debug/flags/set endpoint is enabled or not")

	dirPathWatch = flag.String("dflag_dir_path", "/tmp/foobar", "path to dir to watch updates from.")

	staticInt = flag.Int("example_my_static_int", 1337, "Something integery here.")

	// With generics typing.
	dynStr = dflag.Dyn(flag.CommandLine, "example_my_dynamic_string", "initial_value", "Something interesting here.")
	dynInt = dflag.Dyn(flag.CommandLine, "example_my_dynamic_int", int64(1337), "Something `int`egery here.")
	// This shows the wrong help until 1.20 and https://github.com/golang/go/issues/53473 workaround is removed.
	dynBool1 = dflag.Dyn(flag.CommandLine, "example_bool1", false, "Something true... or false. Starting false.")
	// Or explicit.
	dynBool2 = dflag.DynBool(flag.CommandLine, "example_bool2", true, "Something true... or false. Starting true.")
	// Or new style.
	dynBool3 = dflag.NewBool(true, "defined in 2 steps... Starting true.")
	dynStr2  = dflag.New("starting string value", "explanation of the config variable")
	// This is an example of a dynamically-modifiable JSON flag of an arbitrary type.
	dynJSON = dflag.DynJSON(
		flag.CommandLine,
		"example_my_dynamic_json",
		&exampleConfig{
			Policy: "allow",
			Rate:   50,
			Entries: []*exampleEntry{
				{Name: "foobar", Allowed: true},
			},
		},
		"An arbitrary JSON struct.")
	dynArray  = dflag.New([]string{"z", "b", "a"}, "An array of strings (comma separated)")
	dynSet    = dflag.New(sets.New("z", "b", "a"), "An set of strings (comma separated)")
	dynBinary = dflag.Dyn(flag.CommandLine, "example_binary", []byte{0x00, 0x01, 0x02}, "A binary value")
)

func main() {
	dflag.FlagBool("example_bool3", dynBool3)
	dflag.Flag("example_str2", dynStr2)
	dflag.Flag("example_array", dynArray)
	dflag.Flag("example_set", dynSet)
	dynloglevel.LoggerFlagSetup()
	flag.Parse()
	u, err := configmap.Setup(flag.CommandLine, *dirPathWatch)
	if err != nil {
		log.Fatalf("Failed setting up an updater %v", err)
	}
	defer u.Stop() //nolint:errcheck // just quick example code
	var dflagEndpoint *endpoint.FlagsEndpoint
	if *hasSetFlag {
		setURL := "/debug/flags/set"
		dflagEndpoint = endpoint.NewFlagsEndpoint(flag.CommandLine, setURL)
		http.HandleFunc(setURL, dflagEndpoint.SetFlag)
	} else {
		dflagEndpoint = endpoint.NewFlagsEndpoint(flag.CommandLine, "")
	}
	http.HandleFunc("/debug/flags", dflagEndpoint.ListFlags)
	http.HandleFunc("/", handleDefaultPage)

	addr := fmt.Sprintf("%s:%d", *listenHost, *listenPort)
	log.Infof("Serving at: %v", addr)
	if err := http.ListenAndServe(addr, http.DefaultServeMux); err != nil { //nolint:gosec // just a quick and dirty example
		log.FErrf("Failed serving: %v", err)
		return
	}
	log.Infof("Done, bye.")
}

var defaultPage = template.Must(template.New("default_page").Parse(
	`
<html><head>
<title>Example Server</title>
<link href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.4/css/bootstrap.css" rel="stylesheet">

</head>
<body>
</body>
	<h1>String value {{ .DynString }}</h1>
	<h1>I'm such a number {{ .DynInt }}</h1>
	<h1>Default policy: {{ .Policy }} and number of entries {{ .NumEntries }}.
</html>
`))

func handleDefaultPage(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusOK)
	resp.Header().Add("Content-Type", "text/html")

	actualJSON, _ := dynJSON.Get().(*exampleConfig)
	err := defaultPage.Execute(resp, map[string]any{
		"DynString":  dynStr.Get(),
		"DynInt":     dynInt.Get(),
		"Policy":     actualJSON.Policy,
		"NumEntries": len(actualJSON.Entries),
	})
	if err != nil {
		panic(err)
	}
}

type exampleConfig struct {
	Policy  string          `json:"policy"`
	Rate    int             `json:"rate"`
	Entries []*exampleEntry `json:"entries"`
}

type exampleEntry struct {
	Name    string `json:"entry"`
	Allowed bool   `json:"bool"`
}
