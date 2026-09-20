# Keybindings

Volt's terminal interface uses contextual, Vim-inspired keybindings. Press `?`
outside a text editor to see shortcuts for the focused panel, or press `F1`
anywhere to see every shortcut - the in-app help is generated from the same
action registry this page is written from, so the two never drift apart.

For what these actions actually *do* - Groups/Servers/variables, OpenAPI
import, params, curl export, history/diff, the cookie/auth jar, and
WebSocket - see the [Features guide](features.md). This page only tracks the
keys.

## Key notation

- `Ctrl+key` means hold Control while pressing the key.
- `Alt+key` means Alt on Linux and Windows, or Option on macOS.
- `Esc Esc` means press Escape twice within 750 milliseconds.
- Uppercase bindings such as `G` require Shift.

## Global

| Keys | Action |
|---|---|
| `Ctrl+C` | Quit immediately and cancel an active load test |
| `Esc Esc` | Quit after a deliberate double press |
| `F1` | Show all shortcuts |
| `Ctrl+E` / `Alt+H` | Focus the previous panel |
| `Ctrl+W` / `Alt+L` | Focus the next panel |
| `Ctrl+G` | Open Groups & Servers |
| `Ctrl+N` | Open the interactive OpenAPI import dialog |
| `Ctrl+K` | Open the WebSocket pane |

The first `Esc` closes whatever modal is open (Help, Import, the WebSocket
pane) or returns focus to the sidebar and displays a quit prompt. A different
key or the 750-millisecond timeout cancels the quit sequence.

`q` never quits Volt. It is inserted normally while editing text, does nothing
in non-editing panels, and closes the Help/Groups overlay when one is open.

`Ctrl+E` and `Ctrl+W` work with classic terminal input and are available in
every panel, including while a request field is focused.

### Alt on macOS

macOS calls the Alt key `Option`. The optional panel aliases are therefore
`Option+H` and `Option+L`.

Volt matches Bubble Tea v2's structured keystroke data, so an Option event with
associated macOS text such as `˙` or `¬` still matches its physical `h` or `l`
key. If the terminal sends only text and omits the Option modifier entirely,
use the terminal-independent `Ctrl+E` and `Ctrl+W` bindings.

## Sidebar

| Keys | Action |
|---|---|
| `Enter` | Open the selected saved request |
| `j` | Select the next request |
| `k` | Select the previous request |
| `[count]j` | Move down by a count, wrapping through the list |
| `[count]k` | Move up by a count, wrapping through the list |
| `g` | Select the first request |
| `G` | Select the last request |
| `d` | Delete the selected request |
| `?` | Show sidebar shortcuts |

## Request panel

| Keys | Action |
|---|---|
| `Tab` | Focus the next field |
| `Shift+Tab` | Focus the previous field |
| `h` / `←` | Select the previous HTTP method |
| `l` / `→` | Select the next HTTP method |
| `Alt+Enter` (`Option+Enter` on macOS) / `Ctrl+P` | Send the request |
| `Enter` | Send when the `→ Send` control is focused; otherwise edits the field or accepts a URL suggestion |
| `Ctrl+S` | Save the request |
| `Ctrl+L` | Toggle load-test mode |
| `Ctrl+R` | Reset the whole request (method, URL, name, body, headers, params) |
| `Ctrl+X` | Clear path/query/body/array param values without resetting the rest |
| `Ctrl+Y` | Copy the fully-resolved request as a runnable `curl` command |
| `PgUp` / `PgDn` | Scroll the field list when it's taller than the panel |
| `?` | Show request shortcuts when not editing |

Plain `Enter` remains available to edit content or accept suggestions while a
text field is focused - `Enter` or `→` also accepts a URL autocomplete
suggestion. Arrow keys remain available to text editors. Request bodies with
no imported schema are edited as raw text, so JSON, XML, GraphQL, and other
formats do not require key/value conversion; an imported schema instead
renders each field as its own input, enum picker, or array-count control (see
[Params](features.md#filling-in-a-request) in the Features guide).

While a text field is focused, printable shortcut characters such as `?`, `q`,
`h`, and `l` are inserted into the field instead of triggering panel actions.

## Response panel

| Keys | Action |
|---|---|
| `h` | Select the previous response tab |
| `l` | Select the next response tab |
| `1`–`5` | Jump directly to a response tab (Body/Headers/Timing/Request/WebSocket) |
| `j` | Scroll down |
| `k` | Scroll up |
| `Ctrl+D` | Scroll down half a page |
| `Ctrl+U` | Scroll up half a page |
| `gg` | Go to the top |
| `G` | Go to the bottom |
| `g{n}g` | Go to line `n` |
| `r` | Toggle raw vs. formatted body |
| `w` | Toggle line wrapping |
| `s` | Save the raw response body to a file |
| `y` | Copy the response body |
| `z` | Zoom the response panel to the full right column |
| `t` | Pick a syntax highlighting style (with live preview) |
| `/` | Search the current tab (regex, incremental) |
| `n` / `b` | Next / previous search match |
| `N` / `P` | Next / previous match, re-anchored to what's on screen |
| `c` | Remember this response's cookies and auth header for its host |
| `v` | View or clear saved cookies/auth headers for this host |
| `H` | Browse past sends this session; `x` on an entry diffs it against the current response |
| `Ctrl+X` | Cancel an active load test |
| `?` | Show response shortcuts |

The 5th (WebSocket) tab only exists once there's something to show it - see
[WebSocket](features.md#websocket) in the Features guide.

See [History and diffing responses](features.md#history-and-diffing-responses)
and [Cookies and auth headers](features.md#cookies-and-auth-headers-the-jar)
for what `H`/`x` and `c`/`v` actually do.

## WebSocket pane (`Ctrl+K`)

| Keys | Action |
|---|---|
| `Tab` / `Shift+Tab` | Cycle URL → Connect/Disconnect → message field |
| `Enter` | Connect/disconnect (URL or button focused) or send (message field focused, while connected) |
| `Esc` | Hide the pane - a live session keeps running in the background |

## Groups & Servers (`Ctrl+G`)

| Keys | Action |
|---|---|
| `Tab` | Switch between the Groups list and the selected group's Servers list |
| `n` | Create a group (Groups focused) or a server in the selected group (Servers focused) |
| `r` | Rename the selected group or server |
| `d` | Delete the selected group (and its requests) or server (removed from every group) |
| `u` | Edit the selected server's base URL |
| `H` | Edit the selected server's default headers |
| `V` | Edit the selected server's variables (see [Variables](features.md#groups-servers-and-variables)) |
| `q` / `Esc` | Close |

## Import an OpenAPI spec (`Ctrl+N`)

| Keys | Action |
|---|---|
| `Tab` / `Shift+Tab` | Cycle spec path → group → base URL → force toggle |
| `Space` | Toggle "force" (overwrite already-imported requests) while it's focused |
| `Enter` | Import |
| `Esc` | Close |

See [Importing an OpenAPI spec](features.md#importing-an-openapi-spec) for
what each field does and how it maps to `volt import`'s flags.

## Help overlay

| Keys | Action |
|---|---|
| `h` | Show the previous help section |
| `l` | Show the next help section |
| `1`–`7` | Jump directly to a help section |
| `q` or `?` | Close help |
| `Esc` | Close help and arm the double-Escape quit sequence |

The in-application help is generated from the same action registry used for
input dispatch. When a binding changes in code, its displayed help and
contextual binding tests change with it.
