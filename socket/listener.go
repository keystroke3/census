// Package socket provides network communication capabilities for the census daemon.
// It handles TCP and Unix socket listeners, message handling, and remote query
// execution. The package supports both server-side listening (via TCPListen and
// UnixListen) and client-side communication (via RemoteQuery and dialCommand).
package socket

import (
	"bytes"
	"census/index"
	"census/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func Listen(listener net.Listener) {
	defer listener.Close()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		if sig != nil {
			listener.Close()
			slog.Info("closing open connections...")
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if _, ok := err.(net.Error); ok {
				slog.Info("quit signal received, shutting down...", "error", err)
				break
			}
			if err != io.EOF {
				slog.Error("unable to accept connections", "error", err)
				break
			}
		}
		go handleMessage(conn)
	}

}

func sendResponse(conn net.Conn, resp *types.NetResponse) error {
	respb, err := json.Marshal(resp)
	if err != nil {
		slog.Error("unable to marshal response", "error", err, "response", resp)
		return err
	}
	_, err = conn.Write(respb)
	if err != nil {
		slog.Error("unable to write response", "error", err, "response", resp)
		return err
	}
	return nil
}

func handleMessage(conn net.Conn) {
	defer conn.Close()
	slog.Info("handling message", "address", conn.RemoteAddr())
	cmd := &types.Command{}
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		slog.Error("unable to read from socket connection", "error", err)
		return
	}
	resp := types.NetResponse{}
	rcmd := bytes.Trim(buf, "\x00")
	err = json.Unmarshal(rcmd, cmd)
	if err != nil {
		slog.Error("unable to unmarshal raw command", "error", err, "command", rcmd)
		resp.Error = fmt.Sprintf("unable to read command. %v", err)
		respb, merr := json.Marshal(resp)
		if merr != nil {
			slog.Error("unable to marshal response", "error", merr, "response", resp)
			return
		}
		conn.Write(respb)
		return
	}

	if cmd.StopServer {
		slog.Info("received shutdown signal. Quitting...")
		resp.Ack = true
		sendResponse(conn, &resp)
		syscall.Exit(0)
	}

	err = localizeHomePaths(cmd)
	if err != nil {
		resp.Error = err.Error()
		respb, merr := json.Marshal(resp)
		if merr != nil {
			slog.Error("unable to marshal response", "error", merr, "response", resp)
			return
		}
		conn.Write(respb)
		return
	}
	result, err := index.Query(cmd, nil)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Paths = result
	}
	sendResponse(conn, &resp)

}

func localizeHomePaths(args *types.Command) error {
	cleanPaths := []string{}
	for _, path := range args.Paths {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to determine local home directory")
		}
		if strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
			cleanPaths = append(cleanPaths, filepath.Join(home, path[2:]))
		} else if path == "~" {
			cleanPaths = append(cleanPaths, home)
		} else {
			cleanPaths = append(cleanPaths, path)
		}
	}
	args.Paths = cleanPaths
	home, err := os.UserHomeDir()
	if err == nil {
		for i, pfx := range args.Trim {
			if strings.HasPrefix(pfx, "~"+string(os.PathSeparator)) {
				args.Trim[i] = filepath.Join(home, pfx[2:])
			} else if pfx == "~" {
				args.Trim[i] = home
			}
		}
	}
	return nil
}

func dialCommand(cmnd *types.Command, protocol ...string) (*types.NetResponse, error) {
	proto := "tcp"
	if len(protocol) > 0 {
		proto = protocol[0]
	}
	conn, err := net.Dial(proto, cmnd.Host)
	if err != nil {
		fmt.Println("unable to connect to host", cmnd.Host, "target may be offline")
		return nil, err
	}
	defer conn.Close()
	msg, err := json.Marshal(cmnd)
	if err != nil {
		fmt.Println("unable to marshal command, ", err)
		return nil, err
	}
	conn.Write(msg)
	resp, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}
	var results types.NetResponse
	err = json.Unmarshal(resp, &results)
	if err != nil {
		fmt.Println("Unable to read host response, ", err)
		syscall.Exit(1)
	}
	return &results, nil
}

func StopTCPListen(cmnd *types.Command) {
	results, err := dialCommand(cmnd)
	if err != nil {
		fmt.Println("error shutting down server", err)
		syscall.Exit(1)
	}
	if results.Error != "" {
		if strings.HasPrefix(results.Error, "path not found") {
			fmt.Printf(strings.ReplaceAll(results.Error, "path not found", "path not found on "+cmnd.Host))
		} else {
			fmt.Println("host returned an error:", results.Error)
		}
		syscall.Exit(1)
	}
	fmt.Println("host", cmnd.Host, "shut down")
}

func TCPListen(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("unable to listen on address, ", err)
	}
	slog.Info("census http daemon listening on", "address", addr)
	Listen(listener)
}

func UnixListen(file string) {
	checkSocketConflict(file)
	listener, err := net.Listen("unix", file)
	defer os.Remove(file)
	if err != nil {
		log.Fatal("unable to initialize the daemon ", err)
	}
	Listen(listener)
}

func checkSocketConflict(file string) {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatal("unable to check for daemon conflicts, ", err)
	}
	conn, err := net.Dial("unix", file)
	if err != nil {
		os.Remove(file)
		return
	}
	defer conn.Close()
	fmt.Println("another census is already running")
	syscall.Exit(1)

}

func RemoteQuery(args *types.Command) string {
	results, err := dialCommand(args)
	if err != nil {
		fmt.Println("unable to send command", err)
		syscall.Exit(1)
	}
	if results.Error != "" {
		if strings.HasPrefix(results.Error, "path not found") {
			fmt.Printf(strings.ReplaceAll(results.Error, "path not found", "path not found on "+args.Host))
		} else {
			fmt.Println("Remote host returned an error:", results.Error)
		}
		syscall.Exit(1)
	}
	return results.Paths
}
