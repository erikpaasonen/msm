package cmd

import (
	"fmt"

	"github.com/msmhq/msm/internal/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server management commands",
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := server.DiscoverAll(cfg)
		if err != nil {
			return err
		}

		if len(servers) == 0 {
			fmt.Println("No servers found.")
			return nil
		}

		for _, s := range servers {
			status := s.Status()
			fmt.Printf("%s (%s)\n", s.Name, status)
		}
		return nil
	},
}

var serverCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s, err := server.Create(name, cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Created server %q at %s\n", s.Name, s.Path)
		return nil
	},
}

var serverDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := server.Delete(name, cfg); err != nil {
			return err
		}
		fmt.Printf("Deleted server %q\n", name)
		return nil
	},
}

var serverRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName, newName := args[0], args[1]
		if err := server.Rename(oldName, newName, cfg); err != nil {
			return err
		}
		fmt.Printf("Renamed server %q to %q\n", oldName, newName)
		return nil
	},
}

var serverInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Initialize missing config files for a server",
	Long: `Initialize missing configuration files for an existing server.

This is useful when importing a world into an existing server directory.
It will create any missing files:
  - eula.txt (auto-accepted)
  - server.properties (with auto-assigned port)
  - worldstorage directories

Existing files are not overwritten.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		created, err := server.Init(name, cfg)
		if err != nil {
			return err
		}
		if len(created) == 0 {
			fmt.Printf("Server %q already fully initialized\n", name)
		} else {
			fmt.Printf("Initialized server %q:\n", name)
			for _, f := range created {
				fmt.Printf("  - Created %s\n", f)
			}
		}
		return nil
	},
}

func serverAction(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("server name required")
		}

		name := args[0]
		s, err := server.Get(name, cfg)
		if err != nil {
			return err
		}

		now, _ := cmd.Flags().GetBool("now")

		switch action {
		case "start":
			if err := s.Start(); err != nil {
				return err
			}
			if port := s.Port(); port > 0 {
				fmt.Printf("Started server %q on port %d\n", name, port)
			} else {
				fmt.Printf("Started server %q\n", name)
			}

		case "stop":
			if err := s.Stop(now); err != nil {
				return err
			}
			fmt.Printf("Stopped server %q\n", name)

		case "restart":
			if err := s.Restart(now); err != nil {
				return err
			}
			fmt.Printf("Restarted server %q\n", name)

		case "status":
			fmt.Printf("Server %q is %s\n", name, s.Status())

		case "console":
			return s.Console()

		case "connected":
			if !s.IsRunning() {
				fmt.Printf("Server %q is not running\n", name)
				return nil
			}
			fmt.Printf("Server %q is running (use 'msm %s console' to connect)\n", name, name)
		}

		return nil
	}
}

func addServerActionCommands() {
	actions := []struct {
		name   string
		short  string
		hasNow bool
	}{
		{"start", "Start the server", false},
		{"stop", "Stop the server", true},
		{"restart", "Restart the server", true},
		{"status", "Show server status", false},
		{"console", "Attach to server console", false},
		{"connected", "Check if server is running", false},
	}

	for _, a := range actions {
		cmd := &cobra.Command{
			Use:                a.name,
			Short:              a.short,
			Args:               cobra.ExactArgs(1),
			RunE:               serverAction(a.name),
			DisableFlagParsing: false,
		}

		if a.hasNow {
			cmd.Flags().Bool("now", false, "Execute immediately without warning players")
		}

		rootCmd.AddCommand(cmd)
	}
}

var sayCmd = &cobra.Command{
	Use:   "say <server> <message>",
	Short: "Broadcast a message to all players on a server",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		message := args[1:]

		s, err := server.Get(name, cfg)
		if err != nil {
			return err
		}

		if !s.IsRunning() {
			return fmt.Errorf("server %q is not running", name)
		}

		return s.Say(joinArgs(message))
	},
}

var kickCmd = &cobra.Command{
	Use:   "kick <server> <player> [reason]",
	Short: "Kick a player from the server",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		player := args[1]
		var reason string
		if len(args) > 2 {
			reason = joinArgs(args[2:])
		}

		s, err := server.Get(serverName, cfg)
		if err != nil {
			return err
		}

		if !s.IsRunning() {
			return fmt.Errorf("server %q is not running", serverName)
		}

		return s.Kick(player, reason)
	},
}

var cmdCmd = &cobra.Command{
	Use:   "cmd <server> <command>",
	Short: "Send a command to the server console",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		command := joinArgs(args[1:])

		s, err := server.Get(serverName, cfg)
		if err != nil {
			return err
		}

		if !s.IsRunning() {
			return fmt.Errorf("server %q is not running", serverName)
		}

		return s.SendCommand(command)
	},
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}

func init() {
	serverCmd.AddCommand(serverListCmd)
	serverCmd.AddCommand(serverCreateCmd)
	serverCmd.AddCommand(serverDeleteCmd)
	serverCmd.AddCommand(serverRenameCmd)
	serverCmd.AddCommand(serverInitCmd)
	serverCmd.AddCommand(serverConfigCmd)

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(sayCmd)
	rootCmd.AddCommand(kickCmd)
	rootCmd.AddCommand(cmdCmd)

	addServerActionCommands()
}
