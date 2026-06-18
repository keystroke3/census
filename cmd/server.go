package cmd

import (
	"census/socket"
	"census/types"
	"fmt"

	"github.com/spf13/cobra"
)

var runArgs *types.RunArgs

var (
	serverRun  bool
	serverStop bool

	startDaemon bool
	stopDaemon  bool
)

var serveCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the census server or indexer",
	Run: func(cmd *cobra.Command, args []string) {
		if serverStop {
			cliArgs.StopServer = true
			cliArgs.Host = runArgs.Host
			socket.StopTCPListen(cliArgs)
			return
		}
		if cmd.Flags().Changed("port") {
			runArgs.Host = fmt.Sprintf("0.0.0.0:%d", runArgs.Port)
		}
		socket.TCPListen(runArgs.Host)
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the census indexer daemon",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	runArgs = &types.RunArgs{}
	serveCmd.Flags().BoolVar(&serverRun, "run", false, "starts the census server")
	serveCmd.Flags().BoolVar(&serverStop, "stop", false, "stop the running census server")
	serveCmd.Flags().StringVarP(&runArgs.Host, "address", "a", "0.0.0.0:6450", "address for the server to listen on (default 0.0.0.0:6450)")
	serveCmd.Flags().IntVarP(&runArgs.Port, "port", "p", 6450, "port for the server to listen on. This will cause the host to default to 0.0.0.0, (default 6450)")

	startCmd.Flags().StringSliceVarP(&runArgs.Watch, "watch", "w", nil, "path(s) for the indexer to watch")
	startCmd.Flags().StringSliceVarP(&runArgs.Ignore, "ignore", "i", nil, "path(s) for the indexer to ignore")
	startCmd.Flags().StringVarP(&runArgs.IgnoreFile, "ignore-file", "I", "", "path to file containing list of paths to be ignored by the indexer")
	startCmd.Flags().StringVarP(&runArgs.WatchFile, "watch-file", "W", "", "path to file containing list of paths to be watched by the indexer")
	// rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(serveCmd)
}
