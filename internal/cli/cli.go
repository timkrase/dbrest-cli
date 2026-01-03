package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/timkrase/deutsche-bahn-skill/internal/api"
	"github.com/timkrase/deutsche-bahn-skill/internal/format"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

type OutputMode int

const (
	OutputHuman OutputMode = iota
	OutputPlain
	OutputJSON
)

// Runner wires dependencies for CLI execution.
type Runner struct {
	Out       io.Writer
	Err       io.Writer
	Getenv    func(string) string
	NewClient func(cfg api.Config) (api.Clienter, error)
	Version   string
}

// Run executes the CLI with the provided args and returns an exit code.
func Run(args []string, runner Runner) int {
	out := runner.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := runner.Err
	if errOut == nil {
		errOut = os.Stderr
	}
	getenv := runner.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	newClient := runner.NewClient
	if newClient == nil {
		newClient = func(cfg api.Config) (api.Clienter, error) {
			return api.NewClient(cfg)
		}
	}

	fs := flag.NewFlagSet("dbrest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		helpFlag   bool
		version    bool
		jsonOutput bool
		plain      bool
		baseURL    string
		timeoutStr string
		verbose    bool
	)

	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")
	fs.BoolVar(&version, "version", false, "Show version")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.BoolVar(&plain, "plain", false, "Output stable, line-based text")
	fs.BoolVar(&verbose, "verbose", false, "Print request details to stderr")
	fs.StringVar(&baseURL, "base-url", envOrDefault(getenv, "DBREST_BASE_URL", "https://v6.db.transport.rest"), "API base URL")
	fs.StringVar(&timeoutStr, "timeout", envOrDefault(getenv, "DBREST_TIMEOUT", "10s"), "HTTP timeout (e.g. 10s, 1m)")

	fs.Usage = func() {
		printUsage(errOut)
	}

	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printUsage(errOut)
		return exitUsage
	}

	if helpFlag {
		printUsage(out)
		return exitOK
	}

	if version {
		_, _ = fmt.Fprintln(out, runner.Version)
		return exitOK
	}

	if jsonOutput && plain {
		_, _ = fmt.Fprintln(errOut, "--json and --plain are mutually exclusive")
		return exitUsage
	}

	mode := OutputHuman
	if plain {
		mode = OutputPlain
	}
	if jsonOutput {
		mode = OutputJSON
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "invalid --timeout: %v\n", err)
		return exitUsage
	}

	if fs.NArg() == 0 {
		printUsage(errOut)
		return exitUsage
	}

	client, err := newClient(api.Config{
		BaseURL:   baseURL,
		Timeout:   timeout,
		UserAgent: "dbrest/" + strings.TrimSpace(runner.Version),
	})
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitError
	}

	cmd := fs.Arg(0)
	cmdArgs := fs.Args()[1:]

	switch cmd {
	case "help":
		return runHelp(cmdArgs, out, errOut)
	case "locations":
		return runLocations(cmdArgs, out, errOut, client, mode, verbose)
	case "departures":
		return runDepartures(cmdArgs, out, errOut, client, mode, verbose)
	case "arrivals":
		return runArrivals(cmdArgs, out, errOut, client, mode, verbose)
	case "journeys":
		return runJourneys(cmdArgs, out, errOut, client, mode, verbose)
	case "trip":
		return runTrip(cmdArgs, out, errOut, client, mode, verbose)
	case "radar":
		return runRadar(cmdArgs, out, errOut, client, mode, verbose)
	case "request":
		return runRequest(cmdArgs, out, errOut, client, mode, verbose)
	default:
		_, _ = fmt.Fprintf(errOut, "unknown command: %s\n", cmd)
		printUsage(errOut)
		return exitUsage
	}
}

func runHelp(args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		printUsage(out)
		return exitOK
	}
	switch args[0] {
	case "locations":
		printLocationsUsage(out)
	case "departures":
		printDeparturesUsage(out)
	case "arrivals":
		printArrivalsUsage(out)
	case "journeys":
		printJourneysUsage(out)
	case "trip":
		printTripUsage(out)
	case "radar":
		printRadarUsage(out)
	case "request":
		printRequestUsage(out)
	default:
		_, _ = fmt.Fprintf(errOut, "unknown command: %s\n", args[0])
		printUsage(errOut)
		return exitUsage
	}
	return exitOK
}

type paramList []string

func (p *paramList) String() string {
	return strings.Join(*p, ",")
}

func (p *paramList) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("param must be key=value")
	}
	*p = append(*p, value)
	return nil
}

type floatFlag struct {
	value float64
	set   bool
}

func (f *floatFlag) String() string {
	return strconv.FormatFloat(f.value, 'f', -1, 64)
}

func (f *floatFlag) Set(value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	f.value = parsed
	f.set = true
	return nil
}

func addParams(values url.Values, params []string) error {
	for _, item := range params {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return fmt.Errorf("invalid --param %q (expected key=value)", item)
		}
		values.Add(parts[0], parts[1])
	}
	return nil
}

func runLocations(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("locations", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		query     string
		results   int
		fuzzy     bool
		stops     bool
		addresses bool
		poi       bool
		params    paramList
		helpFlag  bool
	)

	fs.StringVar(&query, "query", "", "Search query")
	fs.IntVar(&results, "results", 10, "Maximum number of results")
	fs.BoolVar(&fuzzy, "fuzzy", true, "Enable fuzzy search")
	fs.BoolVar(&stops, "stops", true, "Include stops and stations")
	fs.BoolVar(&addresses, "addresses", true, "Include addresses")
	fs.BoolVar(&poi, "poi", true, "Include points of interest")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printLocationsUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printLocationsUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printLocationsUsage(out)
		return exitOK
	}
	if query == "" && fs.NArg() > 0 {
		query = fs.Arg(0)
	}
	if strings.TrimSpace(query) == "" {
		_, _ = fmt.Fprintln(errOut, "missing --query")
		printLocationsUsage(errOut)
		return exitUsage
	}

	values := url.Values{}
	values.Set("query", query)
	values.Set("results", strconv.Itoa(results))
	values.Set("fuzzy", strconv.FormatBool(fuzzy))
	values.Set("stops", strconv.FormatBool(stops))
	values.Set("addresses", strconv.FormatBool(addresses))
	values.Set("poi", strconv.FormatBool(poi))
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	return runRequestWithFormatter(out, errOut, client, "/locations", values, mode, verbose, format.LocationsPlain)
}

func runDepartures(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("departures", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		stop      string
		when      string
		duration  int
		results   int
		direction string
		params    paramList
		helpFlag  bool
	)

	fs.StringVar(&stop, "stop", "", "Stop/station id")
	fs.StringVar(&when, "when", "", "Departure time (ISO 8601)")
	fs.IntVar(&duration, "duration", 0, "Search window in minutes")
	fs.IntVar(&results, "results", 0, "Maximum number of results")
	fs.StringVar(&direction, "direction", "", "Direction filter (station id)")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printDeparturesUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printDeparturesUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printDeparturesUsage(out)
		return exitOK
	}
	if stop == "" && fs.NArg() > 0 {
		stop = fs.Arg(0)
	}
	if strings.TrimSpace(stop) == "" {
		_, _ = fmt.Fprintln(errOut, "missing --stop")
		printDeparturesUsage(errOut)
		return exitUsage
	}

	values := url.Values{}
	if when != "" {
		values.Set("when", when)
	}
	if duration > 0 {
		values.Set("duration", strconv.Itoa(duration))
	}
	if results > 0 {
		values.Set("results", strconv.Itoa(results))
	}
	if direction != "" {
		values.Set("direction", direction)
	}
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	path := "/stops/" + url.PathEscape(stop) + "/departures"
	return runRequestWithFormatter(out, errOut, client, path, values, mode, verbose, format.StopoversPlain)
}

func runArrivals(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("arrivals", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		stop      string
		when      string
		duration  int
		results   int
		direction string
		params    paramList
		helpFlag  bool
	)

	fs.StringVar(&stop, "stop", "", "Stop/station id")
	fs.StringVar(&when, "when", "", "Arrival time (ISO 8601)")
	fs.IntVar(&duration, "duration", 0, "Search window in minutes")
	fs.IntVar(&results, "results", 0, "Maximum number of results")
	fs.StringVar(&direction, "direction", "", "Direction filter (station id)")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printArrivalsUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printArrivalsUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printArrivalsUsage(out)
		return exitOK
	}
	if stop == "" && fs.NArg() > 0 {
		stop = fs.Arg(0)
	}
	if strings.TrimSpace(stop) == "" {
		_, _ = fmt.Fprintln(errOut, "missing --stop")
		printArrivalsUsage(errOut)
		return exitUsage
	}

	values := url.Values{}
	if when != "" {
		values.Set("when", when)
	}
	if duration > 0 {
		values.Set("duration", strconv.Itoa(duration))
	}
	if results > 0 {
		values.Set("results", strconv.Itoa(results))
	}
	if direction != "" {
		values.Set("direction", direction)
	}
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	path := "/stops/" + url.PathEscape(stop) + "/arrivals"
	return runRequestWithFormatter(out, errOut, client, path, values, mode, verbose, format.StopoversPlain)
}

func runJourneys(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("journeys", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		from      string
		to        string
		via       string
		departure string
		arrival   string
		results   int
		transfers int
		params    paramList
		helpFlag  bool
	)

	fs.StringVar(&from, "from", "", "Origin station/location id or name")
	fs.StringVar(&to, "to", "", "Destination station/location id or name")
	fs.StringVar(&via, "via", "", "Via station/location id or name")
	fs.StringVar(&departure, "departure", "", "Departure time (ISO 8601)")
	fs.StringVar(&arrival, "arrival", "", "Arrival time (ISO 8601)")
	fs.IntVar(&results, "results", 0, "Maximum number of results")
	fs.IntVar(&transfers, "transfers", 0, "Maximum number of transfers")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printJourneysUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printJourneysUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printJourneysUsage(out)
		return exitOK
	}
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		_, _ = fmt.Fprintln(errOut, "--from and --to are required")
		printJourneysUsage(errOut)
		return exitUsage
	}
	if departure != "" && arrival != "" {
		_, _ = fmt.Fprintln(errOut, "--departure and --arrival are mutually exclusive")
		return exitUsage
	}

	values := url.Values{}
	values.Set("from", from)
	values.Set("to", to)
	if via != "" {
		values.Set("via", via)
	}
	if departure != "" {
		values.Set("departure", departure)
	}
	if arrival != "" {
		values.Set("arrival", arrival)
	}
	if results > 0 {
		values.Set("results", strconv.Itoa(results))
	}
	if transfers > 0 {
		values.Set("transfers", strconv.Itoa(transfers))
	}
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	return runRequestWithFormatter(out, errOut, client, "/journeys", values, mode, verbose, format.JourneysPlain)
}

func runTrip(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("trip", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		tripID   string
		lineName string
		params   paramList
		helpFlag bool
	)

	fs.StringVar(&tripID, "id", "", "Trip id")
	fs.StringVar(&lineName, "line-name", "", "Line name filter")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printTripUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printTripUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printTripUsage(out)
		return exitOK
	}
	if tripID == "" && fs.NArg() > 0 {
		tripID = fs.Arg(0)
	}
	if strings.TrimSpace(tripID) == "" {
		_, _ = fmt.Fprintln(errOut, "missing --id")
		printTripUsage(errOut)
		return exitUsage
	}

	values := url.Values{}
	if lineName != "" {
		values.Set("lineName", lineName)
	}
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	path := "/trips/" + url.PathEscape(tripID)
	return runRequestWithFormatter(out, errOut, client, path, values, mode, verbose, format.TripPlain)
}

func runRadar(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("radar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		north    floatFlag
		south    floatFlag
		west     floatFlag
		east     floatFlag
		results  int
		duration int
		params   paramList
		helpFlag bool
	)

	fs.Var(&north, "north", "Northern latitude")
	fs.Var(&south, "south", "Southern latitude")
	fs.Var(&west, "west", "Western longitude")
	fs.Var(&east, "east", "Eastern longitude")
	fs.IntVar(&results, "results", 0, "Maximum number of results")
	fs.IntVar(&duration, "duration", 0, "Timespan in seconds")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printRadarUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printRadarUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printRadarUsage(out)
		return exitOK
	}
	if !north.set || !south.set || !west.set || !east.set {
		_, _ = fmt.Fprintln(errOut, "--north, --south, --west, and --east are required")
		printRadarUsage(errOut)
		return exitUsage
	}

	values := url.Values{}
	values.Set("north", formatFloatArg(north.value))
	values.Set("south", formatFloatArg(south.value))
	values.Set("west", formatFloatArg(west.value))
	values.Set("east", formatFloatArg(east.value))
	if results > 0 {
		values.Set("results", strconv.Itoa(results))
	}
	if duration > 0 {
		values.Set("duration", strconv.Itoa(duration))
	}
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	return runRequestWithFormatter(out, errOut, client, "/radar", values, mode, verbose, format.RadarPlain)
}

func runRequest(args []string, out io.Writer, errOut io.Writer, client api.Clienter, mode OutputMode, verbose bool) int {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		path     string
		params   paramList
		helpFlag bool
	)

	fs.StringVar(&path, "path", "", "API path (e.g. /journeys)")
	fs.Var(&params, "param", "Extra query param key=value (repeatable)")
	fs.BoolVar(&helpFlag, "help", false, "Show help")
	fs.BoolVar(&helpFlag, "h", false, "Show help (shorthand)")

	fs.Usage = func() {
		printRequestUsage(errOut)
	}
	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		printRequestUsage(errOut)
		return exitUsage
	}
	if helpFlag {
		printRequestUsage(out)
		return exitOK
	}
	if path == "" && fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	if strings.TrimSpace(path) == "" {
		_, _ = fmt.Fprintln(errOut, "missing --path")
		printRequestUsage(errOut)
		return exitUsage
	}

	values := url.Values{}
	if err := addParams(values, params); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return exitUsage
	}

	return runRequestRaw(out, errOut, client, path, values, mode, verbose)
}

func runRequestWithFormatter(out io.Writer, errOut io.Writer, client api.Clienter, path string, values url.Values, mode OutputMode, verbose bool, formatter func([]byte, bool) (string, error)) int {
	data, err := fetch(errOut, client, path, values, verbose)
	if err != nil {
		return exitError
	}
	if mode == OutputJSON {
		writeJSON(out, data)
		return exitOK
	}
	withHeader := mode == OutputHuman
	formatted, err := formatter(data, withHeader)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "formatting error: %v\n", err)
		return exitError
	}
	if formatted != "" {
		_, _ = fmt.Fprint(out, formatted)
	}
	return exitOK
}

func runRequestRaw(out io.Writer, errOut io.Writer, client api.Clienter, path string, values url.Values, mode OutputMode, verbose bool) int {
	data, err := fetch(errOut, client, path, values, verbose)
	if err != nil {
		return exitError
	}
	if mode == OutputPlain {
		_, _ = fmt.Fprint(out, string(data))
		if len(data) == 0 || data[len(data)-1] != '\n' {
			_, _ = fmt.Fprintln(out)
		}
		return exitOK
	}
	writeJSON(out, data)
	return exitOK
}

func fetch(errOut io.Writer, client api.Clienter, path string, values url.Values, verbose bool) ([]byte, error) {
	if verbose {
		if urlStr, err := client.URL(path, values); err == nil {
			_, _ = fmt.Fprintf(errOut, "GET %s\n", urlStr)
		}
	}
	data, err := client.Get(context.Background(), path, values)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return nil, err
	}
	return data, nil
}

func writeJSON(out io.Writer, data []byte) {
	_, _ = out.Write(data)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		_, _ = fmt.Fprintln(out)
	}
}

func envOrDefault(getenv func(string) string, key, fallback string) string {
	if getenv == nil {
		return fallback
	}
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func formatFloatArg(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func printUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `dbrest - Deutsche Bahn transport API CLI

USAGE:
  dbrest [global flags] <command> [args]

COMMANDS:
  locations   Search for stations/places/addresses
  departures  List departures for a stop
  arrivals    List arrivals for a stop
  journeys    Find journeys between two locations
  trip        Fetch a trip by id
  radar       List vehicle movements in a bounding box
  request     Perform a raw GET request
  help        Show command help

GLOBAL FLAGS:
  -h, --help           Show help
      --version        Show version
      --json           Output raw JSON
      --plain          Output stable, line-based text
      --base-url       API base URL (default: https://v6.db.transport.rest)
      --timeout        HTTP timeout (default: 10s)
      --verbose        Print request details to stderr

OUTPUT MODES:
  --json   Raw API response JSON
  --plain  Tab-separated columns, no header (request prints raw JSON)

ENV:
  DBREST_BASE_URL   Override the API base URL
  DBREST_TIMEOUT    Override the HTTP timeout

EXAMPLES:
  dbrest locations Berlin
  dbrest departures --stop 8011160 --results 5
  dbrest journeys --from Berlin --to Hamburg --plain

Run 'dbrest help <command>' for command-specific help.`)
}

func printLocationsUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest locations --query <text> [flags]
  dbrest locations <text> [flags]

FLAGS:
  --query        Search query (required)
  --results      Maximum number of results (default: 10)
  --fuzzy        Enable fuzzy search (default: true)
  --stops        Include stops and stations (default: true)
  --addresses    Include addresses (default: true)
  --poi          Include points of interest (default: true)
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

EXAMPLE:
  dbrest locations Berlin`)
}

func printDeparturesUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest departures --stop <id> [flags]
  dbrest departures <id> [flags]

FLAGS:
  --stop         Stop/station id (required)
  --when         Departure time (ISO 8601)
  --duration     Search window in minutes
  --results      Maximum number of results
  --direction    Direction filter (station id)
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

EXAMPLE:
  dbrest departures 8011160 --results 5`)
}

func printArrivalsUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest arrivals --stop <id> [flags]
  dbrest arrivals <id> [flags]

FLAGS:
  --stop         Stop/station id (required)
  --when         Arrival time (ISO 8601)
  --duration     Search window in minutes
  --results      Maximum number of results
  --direction    Direction filter (station id)
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

EXAMPLE:
  dbrest arrivals 8011160 --when "2024-02-01T08:00:00+01:00"`)
}

func printJourneysUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest journeys --from <id|name> --to <id|name> [flags]

FLAGS:
  --from         Origin station/location id or name (required)
  --to           Destination station/location id or name (required)
  --via          Via station/location id or name
  --departure    Departure time (ISO 8601)
  --arrival      Arrival time (ISO 8601)
  --results      Maximum number of results
  --transfers    Maximum number of transfers
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

EXAMPLE:
  dbrest journeys --from Berlin --to Hamburg --results 3`)
}

func printTripUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest trip --id <trip-id> [flags]
  dbrest trip <trip-id> [flags]

FLAGS:
  --id           Trip id (required)
  --line-name    Line name filter
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

EXAMPLE:
  dbrest trip 1|2|... --line-name "ICE 1000"`)
}

func printRadarUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest radar --north <lat> --south <lat> --west <lon> --east <lon> [flags]

FLAGS:
  --north        Northern latitude (required)
  --south        Southern latitude (required)
  --west         Western longitude (required)
  --east         Eastern longitude (required)
  --results      Maximum number of results
  --duration     Timespan in seconds
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

EXAMPLE:
  dbrest radar --north 52.6 --south 52.4 --west 13.2 --east 13.5 --results 50`)
}

func printRequestUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, `USAGE:
  dbrest request --path <path> [flags]
  dbrest request <path> [flags]

FLAGS:
  --path         API path (required)
  --param        Extra query param key=value (repeatable)
  -h, --help     Show help

NOTE:
  --plain prints raw JSON for this command.

EXAMPLE:
  dbrest request /stations --param query=Berlin --json`)
}
