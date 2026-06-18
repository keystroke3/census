// Package cmd provides handles and parses user comandline arguments and invokes the relevant
// functions.
package cmd

import (
	"census/index"
	"census/socket"
	"census/types"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var cliArgs *types.Command

func remotizePath(path string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine local home directory")
	}
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return path, nil
	}
	return filepath.Join("~", rel), nil
}

func remotizeHomePaths(paths []string) ([]string, error) {
	cleanPaths := []string{}
	for _, path := range paths {
		p, err := remotizePath(path)
		if err != nil {
			return nil, err
		}
		cleanPaths = append(cleanPaths, p)
	}
	return cleanPaths, nil
}

func readPipedPaths() []string {
	f, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("error reading from pipe %s", err)
	}
	if f.Mode()&os.ModeNamedPipe == 0 {
		return nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("error reading from pipe %s", err)
	}
	paths := strings.Split(string(b), "\n")
	cleanPaths := []string{}
	for _, path := range paths {
		if path != "" {
			cleanPaths = append(cleanPaths, path)
		}

	}
	return cleanPaths
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "census",
	Short: "A tool to index, search through and find your files",
	Long: `Census takes census (get it?) of files in a specified directory or
group of directories.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		pipePaths := readPipedPaths()
		if pipePaths != nil {
			cliArgs.Paths = pipePaths
		}
		if !cmd.Flags().Changed("paths") && len(cliArgs.Paths) == 0 {
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
			netPaths, err := remotizeHomePaths(cliArgs.Paths)
			if err != nil {
				fmt.Println("error translating paths", err)
				return
			}
			if cmd.Flags().Changed("trim") && len(cliArgs.Trim) > 0 {
				remotePaths := make([]string, len(cliArgs.Trim))
				for i, path := range cliArgs.Trim {
					p, err := remotizePath(path)
					if err != nil {
						fmt.Println("error parsing trim prefix", err)
					}
					remotePaths[i] = p
				}
				cliArgs.Trim = remotePaths
			}
			cliArgs.Paths = netPaths
			results := socket.RemoteQuery(cliArgs)
			fmt.Println(results)
			return
		}
		results, err := index.Query(cliArgs, nil)
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
	rootCmd.Flags().BoolVarP(&cliArgs.Relative, "relative", "r", false, "trim the root search path out")
	rootCmd.Flags().IntVarP(&cliArgs.Depth, "depth", "D", -1, "How many nested directories to index")
	rootCmd.Flags().StringVarP(&cliArgs.Grep, "grep", "g", "", "show path files matches that match regex pattern")
	rootCmd.Flags().StringVarP(&cliArgs.Vgrep, "vgrep", "v", "", "excludes paths match that match regex pattern")
	rootCmd.Flags().StringVarP(&cliArgs.Gsensitive, "grep-case", "G", "", "like grep but case sensitive Overrides grep")
	rootCmd.Flags().StringVarP(&cliArgs.Vsensitive, "vgrep-case", "V", "", "like vgrep but case sensitive. Overrides vgrep")
	rootCmd.Flags().StringArrayVarP(&cliArgs.Trim, "trim", "t", nil, "remove prefix from each path in the results")
	rootCmd.Flags().StringVar(&cliArgs.Host, "host", "", "address for a remote census instance to use instead of local")

}
