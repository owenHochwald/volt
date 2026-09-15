package cli

import "fmt"

// RunGroupCLI dispatches `volt group <subcommand> ...`.
func RunGroupCLI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: volt group <create|list|rename|set-server|delete> ...")
	}

	db, closeDB, err := openDefaultDB()
	if err != nil {
		return err
	}
	defer closeDB()

	switch args[0] {
	case "create":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt group create <name>")
		}
		group, err := db.GetOrCreateGroup(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("Group %q ready (id=%d)\n", group.Name, group.ID)
		return nil

	case "list":
		groups, err := db.ListGroups()
		if err != nil {
			return err
		}
		if len(groups) == 0 {
			fmt.Println("No groups yet. Create one with: volt group create <name>")
			return nil
		}
		servers, err := db.ListServers()
		if err != nil {
			return err
		}
		serverNames := make(map[int64]string, len(servers))
		for _, s := range servers {
			serverNames[s.ID] = s.Name
		}
		for _, g := range groups {
			server := "(none)"
			if g.ServerID != nil {
				if name, ok := serverNames[*g.ServerID]; ok {
					server = name
				}
			}
			fmt.Printf("  %-20s server: %s\n", g.Name, server)
		}
		return nil

	case "rename":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt group rename <old-name> <new-name>")
		}
		if err := db.RenameGroup(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Renamed group %q to %q\n", args[1], args[2])
		return nil

	case "set-server":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt group set-server <group> <server>")
		}
		if err := db.SetGroupServer(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Group %q now points at server %q\n", args[1], args[2])
		return nil

	case "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt group delete <name>")
		}
		if err := db.DeleteGroup(args[1]); err != nil {
			return err
		}
		fmt.Printf("Deleted group %q\n", args[1])
		return nil

	default:
		return fmt.Errorf("unknown group subcommand: %s", args[0])
	}
}
