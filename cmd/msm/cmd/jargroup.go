package cmd

import (
	"fmt"

	"github.com/msmhq/msm/internal/jar"
	"github.com/msmhq/msm/internal/server"
	"github.com/spf13/cobra"
)

var jargroupCmd = &cobra.Command{
	Use:   "jargroup",
	Short: "Jar group management commands",
}

var jargroupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jar groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := jar.DiscoverAll(cfg)
		if err != nil {
			return err
		}

		if len(groups) == 0 {
			fmt.Println("No jar groups found.")
			return nil
		}

		for _, g := range groups {
			fmt.Printf("%s", g.Name)
			if g.URL != "" {
				fmt.Printf(" (%s)", g.URL)
			}
			fmt.Println()
			for _, f := range g.Files {
				fmt.Printf("  - %s\n", f)
			}
		}
		return nil
	},
}

var jargroupCreateCmd = &cobra.Command{
	Use:   "create <name> <url>",
	Short: "Create a new jar group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, url := args[0], args[1]
		g, err := jar.Create(name, url, cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Created jar group %q with URL %s\n", g.Name, g.URL)
		return nil
	},
}

var jargroupDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a jar group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := jar.Delete(name, cfg); err != nil {
			return err
		}
		fmt.Printf("Deleted jar group %q\n", name)
		return nil
	},
}

var jargroupRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a jar group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName, newName := args[0], args[1]
		if err := jar.Rename(oldName, newName, cfg); err != nil {
			return err
		}
		fmt.Printf("Renamed jar group %q to %q\n", oldName, newName)
		return nil
	},
}

var jargroupChangeURLCmd = &cobra.Command{
	Use:   "changeurl <name> <url>",
	Short: "Change the URL for a jar group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, url := args[0], args[1]
		g, err := jar.Get(name, cfg)
		if err != nil {
			return err
		}
		if err := g.ChangeURL(url); err != nil {
			return err
		}
		fmt.Printf("Changed URL for jar group %q to %s\n", name, url)
		return nil
	},
}

var jargroupGetLatestCmd = &cobra.Command{
	Use:   "getlatest <name>",
	Short: "Download the latest jar for a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		g, err := jar.Get(name, cfg)
		if err != nil {
			return err
		}
		filename, err := g.GetLatest()
		if err != nil {
			return err
		}
		fmt.Printf("Downloaded %s to jar group %q\n", filename, name)
		return nil
	},
}

var jarCmd = &cobra.Command{
	Use:   "jar <server> <jargroup> [file]",
	Short: "Link a server to a jar from a jar group",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		jargroupName := args[1]
		var jarFile string
		if len(args) > 2 {
			jarFile = args[2]
		}

		s, err := server.Get(serverName, cfg)
		if err != nil {
			return err
		}

		if err := jar.LinkJar(s.Path, s.Config.JarPath, jargroupName, jarFile, cfg); err != nil {
			return err
		}

		if jarFile != "" {
			fmt.Printf("Linked server %q to jar %s from group %q\n", serverName, jarFile, jargroupName)
		} else {
			fmt.Printf("Linked server %q to latest jar from group %q\n", serverName, jargroupName)
		}
		return nil
	},
}

func init() {
	jargroupCmd.AddCommand(jargroupListCmd)
	jargroupCmd.AddCommand(jargroupCreateCmd)
	jargroupCmd.AddCommand(jargroupDeleteCmd)
	jargroupCmd.AddCommand(jargroupRenameCmd)
	jargroupCmd.AddCommand(jargroupChangeURLCmd)
	jargroupCmd.AddCommand(jargroupGetLatestCmd)

	rootCmd.AddCommand(jargroupCmd)
	rootCmd.AddCommand(jarCmd)
}
