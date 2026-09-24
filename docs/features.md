# Features guide

This page covers what Volt's features actually do and how to use them:
Groups/Servers/variables, importing an OpenAPI spec, filling in a request,
exporting to curl, inspecting a response (raw/formatted body, syntax style,
search, timing breakdown), history and diffing, the cookie/auth jar, and
WebSocket. For the keys themselves, see the [Keybindings](keybindings.md)
reference - this page links to it instead of repeating every binding.

Load testing has its own section in the [README](../README.md#interactive-load-testing).

## Groups, Servers, and Variables

A **Server** is a saved base URL plus a set of default headers and
variables. A **Group** is a named collection of requests that share one or
more Servers - a request in a group is stored as a bare path (e.g.
`/users/{id}`) and resolved against the group's current server at send time,
so switching environments never means editing every saved request's URL.

**Variables** are `{{name}}` placeholders you can use anywhere in a
request's URL, headers, or body. They're substituted from the *resolved
server's* `Variables` map right before sending. A name with no matching
variable is left in place rather than blanked out, so a typo is visible
instead of silently sending an empty string. Whitespace inside the braces is
ignored (`{{ token }}` and `{{token}}` are the same placeholder).

```
GET {{base}}/users/me
Authorization: Bearer {{token}}
```

### Managing Groups and Servers from the CLI

```bash
# Servers
volt server add staging https://staging.example.com
volt server set-header staging Authorization "Bearer static-token"
volt server set-var staging token "abc123"
volt server list                       # shows header/variable counts
volt server headers staging            # numbered list, for unset-header
volt server vars staging               # numbered list, for unset-var
volt server unset-header staging 1
volt server unset-var staging 1
volt server set-url staging https://staging2.example.com
volt server rename staging stage
volt server delete stage               # also removes it from every group

# Groups
volt group create billing
volt group set-server billing staging  # sets the default AND allows it
volt group add-server billing prod     # allows an additional server, not default
volt group servers billing             # lists allowed servers, marks (default)
volt group remove-server billing prod
volt group rename billing billing-api
volt group list
volt group delete billing-api          # also deletes the group's requests
```

### Managing Groups and Servers from the TUI

Press `Ctrl+G` to open the Groups & Servers manager. `Tab` switches between
the Groups list and the selected group's Servers list; `n`/`r`/`d` create,
rename, and delete whichever list is focused. With a server selected, `u`
edits its base URL, `H` edits its default headers, and `V` edits its
variables (a `key: value`-per-line textarea, saved with `Ctrl+S`).

The in-app manager covers creating a server directly inside a group and
deleting it outright; assigning an *existing* server to an additional group,
or changing which allowed server is the default, is currently CLI-only
(`volt group add-server` / `set-server`).

## Importing an OpenAPI spec

```bash
volt import spec.yaml
volt import spec.yaml -group billing -base-url https://staging.example.com
volt import spec.yaml -group billing -force   # overwrite previously-imported requests
volt import https://api.example.com/openapi.yaml   # fetched over the network instead of a local file
```

- The spec argument can be a local file path or an `http(s)://` URL - a
  service's own published spec endpoint imports directly, with no need to
  download it by hand first.
- `-group` imports into that group (created if it doesn't exist yet) and
  stores each request as a bare path, resolved against the group's server.
- `-base-url` sets that group's server URL. Grouped with no `-base-url` and
  no server yet, Volt adopts the spec's own first `servers[]` entry.
- Without `-group`, URLs are stored absolute; `-base-url` (if given) is
  still saved as a `default` server.
- `-force` overwrites an already-imported request's name/headers/body/param
  schema in place instead of skipping it. Requests are matched by
  group+method+URL, so re-importing the same spec into the same group is
  safe to run repeatedly.
- Import also generates path/query/body param schemas from the spec (see
  [Filling in a request](#filling-in-a-request) below) and prints a summary:
  imported/updated/skipped counts, plus any warnings.

Press `Ctrl+N` for the same import as an interactive modal: spec path or
URL, group, base URL, and a force toggle (`Space`), submitted with `Enter`.

## Filling in a request

A request built by hand has a free-text URL and a raw Body textarea - type
`{name}` directly into the URL and Volt renders a **Path Param** field for
it automatically, re-parsed on every keystroke.

An **imported** request additionally gets:

- **Query Params** - one field per query parameter the spec declared (there's
  no placeholder syntax for these, so they only appear when a schema
  provided them).
- **Body Params** - one input per body field: a plain text input, an enum
  picker for a declared enum (booleans use the same picker), cycled with the
  same `h`/`l` or `←`/`→` used to change the HTTP method. A field the spec
  gave no default for starts on `(not set)` and stays reachable by cycling
  back to it - left out of the request body entirely unless you pick a real
  value, and flagged as a validation error on send if it's required.
- **Array Params** - a body field of `type: array` gets a
  `‹ N item(s) ›` count control (`h`/`l` or `←`/`→` add/removes the last
  item, capped at 20, starting with one). An array of objects renders each
  item as its own named field group; an array of scalars renders each item
  as one plain field.

Fields are always in the same order: Server selector (grouped requests
only) → Name → Method/URL → Path Params → Query Params → Body/Array Params
(or the plain Body textarea, if nothing was imported) → Headers → Send.
`Tab`/`Shift+Tab` (or `↓`/`↑`) cycle through all of them.

## Exporting to curl

`Ctrl+Y` in the request panel copies the request as a runnable `curl`
command - method, sorted headers, and body, all fully resolved (variables
substituted, path/query params filled in) exactly as it would actually be
sent.

## Unix sockets

Any URL field - the request pane's URL, a Server's base URL, and the
WebSocket pane's URL - accepts a `unix://<socket-path>:<request-path>` URL
instead of an ordinary `http(s)://`/`ws(s)://` one, for debugging a local
daemon that listens on a Unix domain socket rather than a TCP port:

```
unix:///var/run/docker.sock:/containers/json
```

dials the socket at `/var/run/docker.sock` and requests `/containers/json`
over it - the same idea as curl's `--unix-socket`, folded into one string
instead of a separate flag/field. The request path defaults to `/` when
omitted entirely (`unix:///var/run/docker.sock` alone, for a bare root
health check).

A Server's base URL can use the same scheme, so every request in a group
resolves against one socket the same way a group already shares one TCP
base URL:

```bash
volt server add docker unix:///var/run/docker.sock
volt group set-server infra docker
```

The WebSocket pane (`Ctrl+K`) accepts the same syntax, for a daemon that
speaks WebSocket over a socket instead of a port.

## Inspecting a response

The response panel has four tabs - Body, Headers, Timing, and (for the
actual request that was sent, params and all) Request:

- `r` toggles the Body tab between formatted/highlighted and raw.
- `w` toggles line wrapping.
- `t` opens a fuzzy-filterable list of syntax styles with a live preview;
  `Enter` applies and persists it, `Esc` cancels.
- `/` opens an incremental, regex-capable search over the current tab (falls
  back to a literal match if the query isn't valid regex); `n`/`b` and
  `N`/`P` step through matches.
- `s` saves the raw response body to a file; `y` copies it.
- `z` zooms the response panel to the full right column, hiding the request
  panel until you leave it.

**Timing** shows the total duration and, for a normal (non-load-test) send,
a DNS/TCP-connect/TLS-handshake/time-to-first-byte breakdown. A reused
keep-alive connection skips DNS/connect/TLS entirely (they only happen once
per connection, not once per request) and the tab says so instead of
showing zeroes.

## History and diffing responses

Every non-load-test send is recorded into an in-memory, session-only history
(cleared on restart, capped at the 100 most recent). Press `H` to browse it:
`Enter` shows a past entry (without re-recording it), `d` deletes the
selected entry, `D` clears everything.

Press `x` on a history entry to diff it against whatever's currently
displayed - a colored `+`/`-`/unchanged line-by-line comparison of status,
duration, headers, and body. `Esc` or switching tabs exits the diff back to
normal browsing.

## Cookies and auth headers (the jar)

Volt can remember a host's cookies and `Authorization` header so later
requests to the same host pick them up automatically:

- `c` in the response panel captures the last response's `Set-Cookie`
  headers and the sent request's `Authorization` header for that host.
- `v` opens a viewer of everything currently saved for the current host;
  `d` removes one entry, `D` clears the host entirely.
- Two Settings toggles (`?`, then the Settings tab) - "auto-remember cookies"
  and "auto-remember auth headers" - do this automatically on every
  response instead of only on demand. Both default off.

The jar is in-memory only and never written to disk.

## WebSocket

`Ctrl+K` opens a WebSocket pane: a URL field, a Connect/Disconnect button, a
timestamped message log (`→` sent, `←` received, `•` connection events), and
a field to compose and send messages while connected. Typing a URL and
pressing `Enter` connects; pressing it again on the same URL/button
disconnects. The URL of the last connection that actually succeeded is
remembered and pre-filled the next time you open the pane (even across
restarts), since a session usually keeps reconnecting to the same address.
`Esc` hides the pane - a connected session keeps running and logging in the
background, so reopening it with `Ctrl+K` picks up right where it was left,
not a fresh reconnect.

The response panel also gets a 5th **WebSocket** tab, but only once there's
something to show there: it stays hidden until you've sent at least one
message from the WebSocket pane's compose field *and* something has come
back since. From then on it keeps growing with every message received,
live, whether or not the WebSocket pane itself is open - a way to watch the
socket's traffic without switching away from whatever request you're
working on.

## Settings

Open the Command Center with `?` (or `F1`), then move to the `SETTINGS` tab
with `h`/`l`:

- **Appearance** - browse built-in (`default`, `adaptive`, `mono`) and any
  custom YAML themes with a live full-interface preview; `Enter` saves. See
  [Themes and Customization](../CUSTOMIZATION.md) for writing your own.
- **Cookies & Auth** - the two auto-remember toggles described above.
