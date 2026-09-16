package cli

import (
	"fmt"
	"strings"
)

// RunGroupCLI dispatches `volt group <subcommand> ...`.
func RunGroupCLI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: volt group <create|list|rename|set-server|add-server|remove-server|servers|delete> ...")
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
			defaultServer := "(none)"
			if g.ServerID != nil {
				if name, ok := serverNames[*g.ServerID]; ok {
					defaultServer = name
				}
			}
			allowed, err := db.ListGroupServers(g.ID)
			if err != nil {
				return err
			}
			allowedNames := make([]string, len(allowed))
			for i, s := range allowed {
				allowedNames[i] = s.Name
			}
			allowedStr := "(none)"
			if len(allowedNames) > 0 {
				allowedStr = strings.Join(allowedNames, ", ")
			}
			fmt.Printf("  %-20s default: %-15s allowed servers: %s\n", g.Name, defaultServer, allowedStr)
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
		fmt.Printf("Group %q now defaults to server %q (also added to its allowed servers)\n", args[1], args[2])
		return nil

	case "add-server":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt group add-server <group> <server>")
		}
		if err := db.AddGroupServer(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Server %q added to group %q's allowed servers\n", args[2], args[1])
		return nil

	case "remove-server":
		if len(args) != 3 {
			return fmt.Errorf("usage: volt group remove-server <group> <server>")
		}
		if err := db.RemoveGroupServer(args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("Server %q removed from group %q's allowed servers\n", args[2], args[1])
		return nil

	case "servers":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt group servers <group>")
		}
		group, err := db.GetGroupByName(args[1])
		if err != nil {
			return err
		}
		if group == nil {
			return fmt.Errorf("group not found: %s", args[1])
		}
		allowed, err := db.ListGroupServers(group.ID)
		if err != nil {
			return err
		}
		if len(allowed) == 0 {
			fmt.Printf("Group %q has no allowed servers yet. Add one with: volt group add-server %s <server>\n", args[1], args[1])
			return nil
		}
		for _, s := range allowed {
			marker := ""
			if group.ServerID != nil && s.ID == *group.ServerID {
				marker = " (default)"
			}
			fmt.Printf("  %-20s %s%s\n", s.Name, s.BaseURL, marker)
		}
		return nil

	case "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: volt group delete <name>")
		}
		deletedRequests, err := db.DeleteGroup(args[1])
		if err != nil {
			return err
		}
		if deletedRequests > 0 {
			fmt.Printf("Deleted group %q and %d request(s) in it\n", args[1], deletedRequests)
		} else {
			fmt.Printf("Deleted group %q\n", args[1])
		}
		return nil

	default:
		return fmt.Errorf("unknown group subcommand: %s", args[0])
	}
}
