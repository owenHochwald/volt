# Keybindings

Volt's terminal interface uses contextual, Vim-inspired keybindings. Press `?`
outside a text editor to see shortcuts for the focused panel, or press `F1`
anywhere to see every shortcut.

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
| `Ctrl+E` | Focus the previous panel |
| `Ctrl+W` | Focus the next panel |
| `Alt+H` | Focus the previous panel (alias) |
| `Alt+L` | Focus the next panel (alias) |

The first `Esc` closes help or returns focus to the sidebar and displays a quit
prompt. A different key or the 750-millisecond timeout cancels the quit
sequence.

`q` never quits Volt. It is inserted normally while editing text, does nothing
in non-editing panels, and closes the help overlay when the overlay is open.

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
| `h` | Select the previous HTTP method |
| `l` | Select the next HTTP method |
| `Alt+Enter` (`Option+Enter` on macOS) | Send the request |
| `Enter` | Send when the `→ Send` control is focused |
| `Ctrl+S` | Save the request |
| `Ctrl+L` | Toggle load-test mode |
| `?` | Show request shortcuts when not editing |

Plain `Enter` remains available to edit content or accept suggestions while a
text field is focused. It submits only when the `→ Send` control is focused.
Arrow keys remain available to text editors. Request bodies are edited as raw
text, so JSON, XML, GraphQL, and other formats do not require key/value
conversion.

While a text field is focused, printable shortcut characters such as `?`, `q`,
`h`, and `l` are inserted into the field instead of triggering panel actions.

## Response panel

| Keys | Action |
|---|---|
| `h` | Select the previous response tab |
| `l` | Select the next response tab |
| `1`–`3` | Jump directly to a response tab |
| `j` | Scroll down |
| `k` | Scroll up |
| `Ctrl+D` | Scroll down half a page |
| `Ctrl+U` | Scroll up half a page |
| `y` | Copy the response body |
| `Ctrl+X` | Cancel an active load test |
| `?` | Show response shortcuts |

## Help overlay

| Keys | Action |
|---|---|
| `h` | Show the previous help section |
| `l` | Show the next help section |
| `1`–`4` | Jump directly to a help section |
| `q` or `?` | Close help |
| `Esc` | Close help and arm the double-Escape quit sequence |

The in-application help is generated from the same action registry used for
input dispatch. When a binding changes in code, its displayed help and
contextual binding tests change with it.
