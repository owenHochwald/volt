package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/app"
	"github.com/owenHochwald/Volt/internal/cli"
	"github.com/owenHochwald/Volt/internal/logging"
	"github.com/owenHochwald/Volt/internal/storage"
)

func main() {
	args, verbose := extractVerboseFlag(os.Args[1:])
	storage.Verbose = verbose
	args, dbPathOverride := extractDBPathFlag(args)
	cli.DBPathOverride = dbPathOverride
	if verbose {
		// File-based (see logging.DefaultLogPath) since stdout/stderr
		// aren't usable once the TUI takes over the terminal - the same
		// -v/--debug flag that turns on goose's migration logging
		// (storage.Verbose) also turns this on, for troubleshooting HTTP
		// request/response handling (see internal/http/client.go).
		if path, err := logging.DefaultLogPath(); err == nil {
			if err := logging.Enable(path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to enable logging: %v\n", err)
			} else {
				defer logging.Close()
				logging.Logf("volt starting (args=%v)", args)
			}
		}
	}

	// If any command line arguments are provided, use CLI mode
	if len(args) > 0 {
		// Special handling for help
		if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
			cli.PrintHelp()
			return
		}

		subcommands := map[string]func([]string) error{
			"import": cli.RunImportCLI,
			"group":  cli.RunGroupCLI,
			"server": cli.RunServerCLI,
		}
		if run, ok := subcommands[args[0]]; ok {
			if err := run(args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// All other args go to bench mode
		// Support both "volt bench ..." and "volt ..." (with bench implied)
		benchArgs := args
		if args[0] == "bench" {
			benchArgs = args[1:]
		}

		config, err := cli.ParseBenchFlags(benchArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		if err := cli.RunBench(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error running benchmark: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// TUI mode
	dbPath := dbPathOverride
	if dbPath == "" {
		var err error
		dbPath, err = storage.DefaultDBPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v", err)
			os.Exit(1)
		}
	}

	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		fmt.Printf("Error connecting to database: %v", err)
		return
	}
	defer store.Close()

	p := tea.NewProgram(app.SetupModel(store))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

// extractVerboseFlag pulls a -v/--debug flag out of args, wherever it
// appears (before or after a subcommand), returning the remaining args and
// whether it was present. Internal logging (currently just goose's
// migration output - see storage.Verbose) is off by default so it doesn't
// clutter the TUI's terminal or a CLI subcommand's plain output; this flag
// re-enables it for troubleshooting.
func extractVerboseFlag(args []string) ([]string, bool) {
	remaining := make([]string, 0, len(args))
	verbose := false
	for _, a := range args {
		switch a {
		case "-v", "--debug", "-debug":
			verbose = true
		default:
			remaining = append(remaining, a)
		}
	}
	return remaining, verbose
}

// extractDBPathFlag pulls a --db/-db path out of args, wherever it appears
// (before or after a subcommand, as "--db path" or "--db=path"), returning
// the remaining args and the path (empty if not given). Lets both the TUI
// and every CLI subcommand (see cli.DBPathOverride) be pointed at a
// database other than the real one at ~/.volt/volt.db - e.g. for trying
// something out without risking real saved requests.
func extractDBPathFlag(args []string) ([]string, string) {
	remaining := make([]string, 0, len(args))
	dbPath := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--db" || a == "-db":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--db="):
			dbPath = strings.TrimPrefix(a, "--db=")
		case strings.HasPrefix(a, "-db="):
			dbPath = strings.TrimPrefix(a, "-db=")
		default:
			remaining = append(remaining, a)
		}
	}
	return remaining, dbPath
}
