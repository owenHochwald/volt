package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/app"
	"github.com/owenHochwald/Volt/internal/cli"
	"github.com/owenHochwald/Volt/internal/storage"
)

func main() {
	// If any command line arguments are provided, use CLI mode
	if len(os.Args) > 1 {
		// Special handling for help
		if os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
			cli.PrintHelp()
			return
		}

		// All other args go to bench mode
		// Support both "volt bench ..." and "volt ..." (with bench implied)
		args := os.Args[1:]
		if os.Args[1] == "bench" {
			args = os.Args[2:]
		}

		config, err := cli.ParseBenchFlags(args)
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(homeDir, ".volt", "volt.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		fmt.Printf("Error connecting to database: %v", err)
		return
	}
	defer store.Close()

	p := tea.NewProgram(app.SetupModel(store), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
