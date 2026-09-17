package cli

import "fmt"

// PrintHelp displays usage information
func PrintHelp() {
	fmt.Println(`Volt - Terminal HTTP Client and Load Tester

USAGE:
  volt             Launch interactive TUI
  volt bench       Run CLI load test
  volt import      Import request URLs from an OpenAPI 3.x spec
  volt group       Manage request groups (one-level folders for saved requests)
  volt server      Manage named servers that groups can point their requests at

GLOBAL FLAGS:
  -v, --debug       Show internal logging (e.g. database migration output),
                     hidden by default. Works anywhere: before or after a
                     subcommand.
  --db <path>       Use this database file instead of the default
                     (~/.volt/volt.db). Works anywhere: before or after a
                     subcommand.

IMPORT FLAGS:
  -group <name>     Import into this request group (created if missing)
  -base-url <url>   Set/override the group's server URL, or (ungrouped) the base
                     URL prefixed onto every imported request

GROUP SUBCOMMANDS:
  volt group create <name>                 Create an empty group
  volt group list                          List groups and their assigned server
  volt group rename <old> <new>            Rename a group
  volt group set-server <group> <server>   Point a group's requests at a server
  volt group delete <name>                 Delete a group AND every request in it

SERVER SUBCOMMANDS:
  volt server add <name> <base-url>        Add (or update) a named server
  volt server list                         List servers
  volt server set-url <name> <base-url>    Change a server's base URL - every
                                            group pointing at it moves with it
  volt server rename <old> <new>           Rename a server
  volt server delete <name>                Delete a server
  volt server set-header <name> <key> <value>  Set (or update) a default header,
                                                merged into every request that
                                                uses this server
  volt server headers <name>               List a server's default headers, numbered
  volt server unset-header <name> <index>  Remove a default header by the number
                                            shown in ` + "`volt server headers`" + `

BENCH FLAGS:
  -url <string>     Target URL (required)
  -c <int>          Number of concurrent connections (default: 50)
  -d <duration>     Test duration, e.g. "30s", "5m" (default: 10s)
  -n <int>          Total number of requests (mutually exclusive with -d)
  -m <string>       HTTP method (default: GET)
  -H <string>       Custom header, repeatable (format: "Key: Value")
  -b <string>       Request body
  -t <duration>     Request timeout (default: 30s)
  -rate <int>       Rate limit (requests/sec, 0 = unlimited)
  -keepalive        Enable HTTP keep-alive (default: true)
  -no-keepalive     Disable HTTP keep-alive
  -q                Quiet mode (minimal output)
  -json             Output results as JSON
  -o <file>         Write results to file

EXAMPLES:
  # Import all endpoints from a local OpenAPI spec into the saved-requests library
  volt import openapi.yaml

  # Import into a group; requests resolve against that group's server, so
  # switching servers later redirects the whole group at once
  volt import -group petstore -base-url http://localhost:3000 openapi.yaml

  # Point the "petstore" group at a different server, e.g. staging vs prod
  volt server add staging https://staging.api.example.com
  volt group set-server petstore staging

  # Import but point requests at a local server instead of the spec's server
  volt import -base-url http://localhost:3000 openapi.json

  # Basic throughput test
  volt bench -url http://localhost:8080 -c 100 -d 30s

  # POST request with custom headers
  volt bench -url http://localhost:8080/api -m POST \
    -b '{"test":true}' -H "Content-Type: application/json"

  # JSON output to file for CI/CD
  volt bench -url http://localhost:8080 -c 50 -d 60s -json -o results.json

  # Rate-limited testing
  volt bench -url http://localhost:8080 -c 10 -d 30s -rate 1000

  # Quiet mode (just final stats)
  volt bench -url http://localhost:8080 -c 100 -n 10000 -q`)
}
