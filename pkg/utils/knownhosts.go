package utils

import (
	"bufio"
	"os"
	"os/user"
	"path/filepath"
)

// ParseKnownHosts reads the user's known_hosts file and returns each line.
func ParseKnownHosts() ([]string, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(usr.HomeDir, ".ssh", "known_hosts")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// WriteKnownHosts writes the provided lines to the known_hosts file inside sshDirectory.
func WriteKnownHosts(lines []string, sshDirectory string) error {
	p, err := GetAndCreateSshDirectory(sshDirectory)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(p, "known_hosts"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}
