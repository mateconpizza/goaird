package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	addrFlag    string
	versionFlag bool
	loggerFlag  string
)

var (
	logger   *slog.Logger
	logFname string
)

// setupLogger setups the logger.
func setupLogger() (*os.File, error) {
	const filePerm = 0o644 // Permissions for new files.
	f, err := os.OpenFile(logFname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	multiWriter := io.MultiWriter(f, os.Stdout)
	logger = slog.New(slog.NewJSONHandler(multiWriter, nil))

	return f, nil
}

// version returns the application version.
func version() string {
	return fmt.Sprintf("%s v%s %s/%s\n", appName, appVersion, runtime.GOOS, runtime.GOARCH)
}

// notify sends a notification using the notify-send command.
func notify(title, message string) error {
	args := []string{
		"notify-send",
		"--app-name=" + appName,
		"--icon=gnome-user-share",
		title,
		message,
	}

	return executeCmd(args...)
}

// openURL opens a URL in the default browser.
func openURL(s string) error {
	args := osArgs()
	if err := executeCmd(append(args, s)...); err != nil {
		return fmt.Errorf("%w: opening in browser", err)
	}

	return notify(appName, "Opening URL: "+s)
}

// executeCmd runs a command with the given arguments and returns an error if
// the command fails.
func executeCmd(arg ...string) error {
	cmd := exec.CommandContext(context.Background(), arg[0], arg[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}

// getEnv retrieves an environment variable.
//
// If the environment variable is not set, returns the default value.
func getEnv(s, def string) string {
	if v, ok := os.LookupEnv(s); ok {
		return v
	}

	return def
}

// expandHomeDir expands the home directory in the given string.
func expandHomeDir(s string) string {
	if strings.HasPrefix(s, "~/") {
		dirname, _ := os.UserHomeDir()
		s = filepath.Join(dirname, s[2:])
	}

	return s
}

// osArgs returns the correct arguments for the OS.
func osArgs() []string {
	// FIX: only support linux
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = append(args, "open")
	case "windows":
		args = append(args, "cmd", "/C", "start")
	default:
		args = append(args, "xdg-open")
	}

	return args
}

// usage prints the usage message.
func usage() {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Usage:  %s v%s [options]\n\n", appName, appVersion))
	sb.WriteString("\tSimple webhook server\n\n")
	sb.WriteString("Options:\n")
	sb.WriteString("  -a, -addr string\n\tHTTP service address (default \":5001\")\n")
	sb.WriteString("  -V, -version\n\tPrint version and exit\n")
	sb.WriteString("  -l, -log string\n\tLog filepath\n")
	sb.WriteString("  -h, -help\n\tPrint this help message\n")
	sb.WriteString("\nFiles:\n")
	sb.WriteString(fmt.Sprintf("\t%s\n", logFname))

	fmt.Fprint(os.Stderr, sb.String())
}

func init() {
	flag.StringVar(&addrFlag, "addr", ":5001", "HTTP service address")
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
	flag.BoolVar(&versionFlag, "V", false, "Print version and exit")
	flag.StringVar(&loggerFlag, "log", "", "Log filepath")
	flag.StringVar(&loggerFlag, "l", "", "Log filepath")
	flag.Usage = usage

	localState := getEnv("XDG_STATE_HOME", expandHomeDir("~/.local/state"))
	logFname = filepath.Join(localState, appName+".json")

	flag.Parse()
	if versionFlag {
		fmt.Print(version())
		os.Exit(0)
	}

	if loggerFlag != "" {
		logFname = loggerFlag
	}
}
