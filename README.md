# dbrest

Query Deutsche Bahn transport data from `v6.db.transport.rest`.

## CLI spec (create-cli)

1. **Name**: `dbrest`
2. **One-liner**: Query Deutsche Bahn transport data from the DB transport REST API.
3. **USAGE**:
   - `dbrest [global flags] <command> [args]`
4. **Subcommands**:
   - `dbrest locations ...`
   - `dbrest departures ...`
   - `dbrest arrivals ...`
   - `dbrest journeys ...`
   - `dbrest trip ...`
   - `dbrest radar ...`
   - `dbrest request ...`
   - `dbrest help [command]`
5. **Global flags**:
   - `-h, --help` show help and ignore other args
   - `--version` print version to stdout
   - `--json` output raw JSON response
   - `--plain` output stable, line-based text (no headers)
   - `--base-url <url>` override API base URL
   - `--timeout <duration>` HTTP timeout (default `10s`)
   - `--verbose` print request URL to stderr
6. **I/O contract**:
   - stdout: command results (`--json` for machine output; default is human text)
   - stderr: diagnostics, errors, usage, verbose request URLs
7. **Exit codes**:
   - `0` success
   - `1` request/formatting error
   - `2` invalid usage
8. **Env/config**:
   - `DBREST_BASE_URL` (flags override)
   - `DBREST_TIMEOUT` (flags override)
   - precedence: flags > env > defaults
9. **Safety rules**:
   - read-only API calls, no prompts, no destructive operations
10. **Examples**:
   - `dbrest locations --query "Berlin"`
   - `dbrest departures --stop 8011160 --results 5`
   - `dbrest arrivals --stop 8011160 --when "2024-02-01T08:00:00+01:00"`
   - `dbrest journeys --from Berlin --to Hamburg --results 3`
   - `dbrest trip --id 1|2|... --line-name ICE 1000`
   - `dbrest radar --north 52.6 --south 52.4 --west 13.2 --east 13.5 --results 50`
   - `dbrest request --path /stations --param query=Berlin --json`

## Commands

Run `dbrest help <command>` for full help. Each subcommand accepts `--param key=value` to pass through unsupported API query params.

## Machine output contract

- `--json` prints the raw API response JSON for all commands.
- `--plain` prints tab-separated, line-based output with no header row. Missing values are `-`.
- `dbrest request --plain` prints raw JSON (same shape as `--json`) because the response is arbitrary.

Stable `--plain` columns by command:

- `locations`: `id`, `name`, `type`, `latitude`, `longitude`, `distance_m`
- `departures`/`arrivals`: `time`, `line`, `direction`, `platform`, `delay`, `status`
- `journeys`: `departure`, `origin`, `arrival`, `destination`, `transfers`
- `trip`: `line`, `stop`, `arrival`, `departure`, `platform`
- `radar`: `line`, `direction`, `latitude`, `longitude`

## Positional shortcuts

These commands accept a positional fallback for their required flag:

- `dbrest locations <query>` (same as `--query`)
- `dbrest departures <stop>` (same as `--stop`)
- `dbrest arrivals <stop>` (same as `--stop`)
- `dbrest trip <id>` (same as `--id`)
- `dbrest request <path>` (same as `--path`)

## Code quality

Quality is enforced via:

- `gofmt` formatting (checked in `scripts/check.sh`)
- `go vet` static analysis
- `go test ./...` unit tests
- `golangci-lint` with `staticcheck`, `errcheck`, `ineffassign`, `unconvert`, `unparam`

Run all checks:

```
./scripts/check.sh
```

## Tests

```
go test ./...
```
