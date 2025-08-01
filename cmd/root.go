/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"census/index"
	"census/socket"
	"census/types"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var cliArgs *types.Command

func remotizeHomePaths(paths []string) ([]string, error) {
	clean_paths := []string{}
	for _, path := range paths {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("unable to determine local home directory")
		}
		if strings.HasPrefix(path, home) {
			clean_paths = append(clean_paths, strings.Replace(path, home, "~", 1))
		} else {
			clean_paths = append(clean_paths, path)
		}
	}
	return clean_paths, nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "census",
	Short: "A tool to index, search through and find your files",
	Long: `Cencus takes census (get it?) of files in a specified directory or
group of directories.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("paths") || len(cliArgs.Paths) == 0 {
			var paths []string
			if len(args) > 0 {
				paths = args
			} else {
				path, err := os.Getwd()
				if err != nil {
					fmt.Println("could not load paths, ", err)
				}
				paths = []string{path}
			}
			cliArgs.Paths = paths
		}
		if cmd.Flags().Changed("host") && cliArgs.Host != "" {
			net_paths, err := remotizeHomePaths(cliArgs.Paths)
			if err != nil {
				fmt.Println("error translating paths", err)
				return
			}
			cliArgs.Paths = net_paths
			results := socket.RemoteQuery(cliArgs)
			fmt.Println(results)
			return
		}
		results, err := index.Query(cliArgs)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(results)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		syscall.Exit(1)
	}
}

func init() {
	cliArgs = &types.Command{}
	rootCmd.Flags().StringSliceVarP(&cliArgs.Paths, "paths", "p", nil, "path(s) to search through")
	rootCmd.Flags().StringSliceVarP(&cliArgs.IgnorePaths, "ignore", "i", nil, "comma separated paths to ignore when searching. can be passed multiple times")
	rootCmd.Flags().BoolVarP(&cliArgs.ShowHidden, "hidden", "H", false, "whether to include hidden (dot) paths and files in search")
	rootCmd.Flags().StringVarP(&cliArgs.EscapeChars, "escape", "e", "", "characters to prepend with a backslash to escape them")
	rootCmd.Flags().BoolVarP(&cliArgs.Quote, "quote", "q", false, "whether wrap each line in double quotes")
	rootCmd.Flags().BoolVarP(&cliArgs.DirMode, "dir", "d", false, "return directories only")
	rootCmd.Flags().IntVarP(&cliArgs.Depth, "depth", "D", -1, "How many nested directories to index")
	rootCmd.Flags().StringVarP(&cliArgs.Grep, "grep", "g", "", "show path files matches that match regex pattern")
	rootCmd.Flags().StringVarP(&cliArgs.Vgrep, "vgrep", "v", "", "excludes paths match that match regex pattern")
	rootCmd.Flags().StringVarP(&cliArgs.Gsensitive, "grep-case", "G", "", "like grep but case sensitive Overrides grep")
	rootCmd.Flags().StringVarP(&cliArgs.Vsensitive, "vgrep-case", "V", "", "like vgrep but case sensitive. Overrides vgrep")
	rootCmd.Flags().StringVar(&cliArgs.Host, "host", "", "address for a remote census instance to use instead of local")

}
