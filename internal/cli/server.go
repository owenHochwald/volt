package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
)

// RunServerCLI dispatches `volt server <subcommand> ...`.
func RunServerCLI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: volt server <add|list|set-url|rename|delete|set-header|headers|unset-header|set-var|vars|unset-var> ...")
	}

	db, closeDB, err := openDefaultDB()
	if err != nil {
		return err
	}
	defer closeDB()

	switch args[0] {
	case "add":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt server add <name> <base-url>")
		}
		server, err := db.UpsertServer(args[1], args[2])
		if err != nil {
			return err
		}
		fmt.Printf("Server %q -> %s (id=%d)\n", server.Name, server.BaseURL, server.ID)
		return nil

	case "list":
		servers, err := db.ListServers()
		if err != nil {
			return err
		}
		if len(servers) == 0 {
			fmt.Println("No servers yet. Add one with: volt server add <name> <base-url>")
			return nil
		}
		for _, s := range servers {
			fmt.Printf("  %-20s %-30s %d header(s), %d variable(s)\n", s.Name, s.BaseURL, len(s.Headers), len(s.Variables))
		}
		return nil

	case "set-url":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt server set-url <name> <base-url>")
		}
		if err := db.UpdateServerURL(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Server %q now points at %s\n", args[1], args[2])
		return nil

	case "rename":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt server rename <old-name> <new-name>")
		}
		if err := db.RenameServer(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Renamed server %q to %q\n", args[1], args[2])
		return nil

	case "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt server delete <name>")
		}
		if err := db.DeleteServer(args[1]); err != nil {
			return err
		}
		fmt.Printf("Deleted server %q\n", args[1])
		return nil

	case "set-header":
		if len(args) != 4 {
			return fmt.Errorf("usage: volt server set-header <name> <key> <value>")
		}
		server, err := getServerOrFail(db, args[1])
		if err != nil {
			return err
		}
		headers := server.Headers
		if headers == nil {
			headers = make(map[string]string)
		}
		headers[args[2]] = args[3]
		if err := db.UpdateServerHeaders(server.Name, headers); err != nil {
			return err
		}
		fmt.Printf("Set header %q on server %q\n", args[2], server.Name)
		return nil

	case "headers":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt server headers <name>")
		}
		server, err := getServerOrFail(db, args[1])
		if err != nil {
			return err
		}
		if len(server.Headers) == 0 {
			fmt.Printf("Server %q has no default headers.\n", server.Name)
			return nil
		}
		for i, key := range sortedKeys(server.Headers) {
			fmt.Printf("  %d. %s = %s\n", i+1, key, server.Headers[key])
		}
		return nil

	case "unset-header":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt server unset-header <name> <index>")
		}
		server, err := getServerOrFail(db, args[1])
		if err != nil {
			return err
		}
		index, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("invalid index %q: must be a number from `volt server headers %s`", args[2], server.Name)
		}
		keys := sortedKeys(server.Headers)
		if index < 1 || index > len(keys) {
			return fmt.Errorf("no header at index %d (server %q has %d header(s))", index, server.Name, len(keys))
		}
		removedKey := keys[index-1]
		delete(server.Headers, removedKey)
		if err := db.UpdateServerHeaders(server.Name, server.Headers); err != nil {
			return err
		}
		fmt.Printf("Removed header %q from server %q\n", removedKey, server.Name)
		return nil

	case "set-var":
		if len(args) != 4 {
			return fmt.Errorf("usage: volt server set-var <name> <key> <value>")
		}
		server, err := getServerOrFail(db, args[1])
		if err != nil {
			return err
		}
		variables := server.Variables
		if variables == nil {
			variables = make(map[string]string)
		}
		variables[args[2]] = args[3]
		if err := db.UpdateServerVariables(server.Name, variables); err != nil {
			return err
		}
		fmt.Printf("Set variable %q on server %q\n", args[2], server.Name)
		return nil

	case "vars":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt server vars <name>")
		}
		server, err := getServerOrFail(db, args[1])
		if err != nil {
			return err
		}
		if len(server.Variables) == 0 {
			fmt.Printf("Server %q has no variables.\n", server.Name)
			return nil
		}
		for i, key := range sortedKeys(server.Variables) {
			fmt.Printf("  %d. %s = %s\n", i+1, key, server.Variables[key])
		}
		return nil

	case "unset-var":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt server unset-var <name> <index>")
		}
		server, err := getServerOrFail(db, args[1])
		if err != nil {
			return err
		}
		index, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("invalid index %q: must be a number from `volt server vars %s`", args[2], server.Name)
		}
		keys := sortedKeys(server.Variables)
		if index < 1 || index > len(keys) {
			return fmt.Errorf("no variable at index %d (server %q has %d variable(s))", index, server.Name, len(keys))
		}
		removedKey := keys[index-1]
		delete(server.Variables, removedKey)
		if err := db.UpdateServerVariables(server.Name, server.Variables); err != nil {
			return err
		}
		fmt.Printf("Removed variable %q from server %q\n", removedKey, server.Name)
		return nil

	default:
		return fmt.Errorf("unknown server subcommand: %s", args[0])
	}
}

// getServerOrFail loads a server by name, turning "not found" into an error
// instead of the nil-with-no-error GetServerByName returns - every
// header/variable subcommand needs the server to exist to do anything
// useful.
func getServerOrFail(db *storage.SQLiteStorage, name string) (*http.Server, error) {
	server, err := db.GetServerByName(name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("server not found: %s", name)
	}
	return server, nil
}

// sortedKeys returns a map's keys in a stable (sorted) order - shared by
// the headers and variables subcommands, so `volt server headers`/`vars`
// print numbers that `volt server unset-header`/`unset-var <name>
// <index>` can reliably refer back to.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
