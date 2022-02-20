package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/pkg/sftp"
)

const (
	OK       = 0
	WARNING  = 1
	CRITICAL = 2
	UNKNOWN  = 3
)

var (
	warning  = flag.Duration("warning", -1, "return WARNING if the lastest snapshot is older than the specified number of hours")
	critical = flag.Duration("critical", -1, "return CRITICAL if the lastest snapshot is older than the specified number of hours")
	repoPath = flag.String("repository", "", "path to restic repository on sftp target")
	sftpHost = flag.String("host", "", "ssh host to be used for sftp connection")
	sftpUser = flag.String("user", "", "ssh user to be used for sftp connection")
	sftpPort = flag.String("port", "22", "ssh port to be used for sftp connection")
)

func parseArgs() error {
	flag.Parse()
	if *warning < 0 {
		return fmt.Errorf("The option 'warning' needs to be set and greater than 0.")
	}
	if *critical < 0 {
		return fmt.Errorf("The option 'critical' needs to be set and greater than 0.")
	}
	if *repoPath == "" {
		return fmt.Errorf("The option 'repository' needs to be set.")
	}
	if *sftpHost == "" {
		return fmt.Errorf("The option 'host' needs to be set.")
	}
	if *sftpUser == "" {
		return fmt.Errorf("The option 'user' needs to be set.")
	}
	if *sftpPort == "" {
		return fmt.Errorf("The option 'port' needs to be a valid port.")
	}
	return nil
}

func getStatusStr(status int) string {
	switch status {
	case OK:
		return "OK"
	case WARNING:
		return "WARNING"
	case CRITICAL:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

func main() {
	rc, msg := mainReturnWithStatus()
	fmt.Printf("%s: %s\n", getStatusStr(rc), msg)
	os.Exit(rc)
}

func mainReturnWithStatus() (int, string) {
	err := parseArgs()
	if err != nil {
		return UNKNOWN, err.Error()
	}

	// Connect to a remote host and request the sftp subsystem via the 'ssh'
	// command. This assumes that passwordless login is correctly configured.
	cmd := exec.Command("ssh", *sftpHost, "-l", *sftpUser, "-p", *sftpPort, "-s", "sftp")

	// send errors from ssh to stderr
	cmd.Stderr = os.Stderr

	// get stdin and stdout
	wr, err := cmd.StdinPipe()
	if err != nil {
		return UNKNOWN, err.Error()
	}
	rd, err := cmd.StdoutPipe()
	if err != nil {
		return UNKNOWN, err.Error()
	}

	// start the process
	if err := cmd.Start(); err != nil {
		return UNKNOWN, err.Error()
	}
	defer cmd.Wait()

	// open the SFTP session
	client, err := sftp.NewClientPipe(rd, wr)
	if err != nil {
		return UNKNOWN, err.Error()
	}
	defer client.Close()

	// get a list of all snapshots in the restic repository
	files, err := client.ReadDir(*repoPath + "/snapshots")
	if err != nil {
		return UNKNOWN, err.Error()
	}

	if len(files) == 0 {
		return CRITICAL, "no snapshots found"
	}

	// sort snapshots by modtime
	sort.Slice(files, func(a, b int) bool {
		return files[b].ModTime().Before(files[a].ModTime())
	})

	age := time.Now().Sub(files[0].ModTime())

	// sanity check
	if age < 0 {
		return CRITICAL, "latest snapshot is in the future"
	}
	msg := fmt.Sprintf("latest snapshot created %s ago", age.Round(time.Second))
	if age > *critical {
		return CRITICAL, msg
	} else if age > *warning {
		return WARNING, msg
	} else {
		return OK, msg
	}
}
