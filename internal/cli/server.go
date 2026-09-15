package cli

import "fmt"

// RunServerCLI dispatches `volt server <subcommand> ...`.
func RunServerCLI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: volt server <add|list|set-url|rename|delete> ...")
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
			fmt.Printf("  %-20s %s\n", s.Name, s.BaseURL)
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

	default:
		return fmt.Errorf("unknown server subcommand: %s", args[0])
	}
}
