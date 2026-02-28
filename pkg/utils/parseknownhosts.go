package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func ParseKnownHosts(path string) ([]models.KnownHostEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []models.KnownHostEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		var entry models.KnownHostEntry
		if strings.HasPrefix(fields[0], "@") {
			if len(fields) < 4 {
				continue
			}
			entry.Marker = fields[0]
			entry.HostPattern = fields[1]
			entry.KeyType = fields[2]
			entry.KeyData = fields[3]
		} else {
			if len(fields) < 3 {
				continue
			}
			entry.Marker = ""
			entry.HostPattern = fields[0]
			entry.KeyType = fields[1]
			entry.KeyData = fields[2]
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func WriteKnownHosts(entries []models.KnownHostEntry, sshDirectory string) error {
	p, err := GetAndCreateSshDirectory(sshDirectory)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(p, "known_hosts"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, entry := range entries {
		var line string
		if entry.Marker != "" {
			line = fmt.Sprintf("%s %s %s %s\n", entry.Marker, entry.HostPattern, entry.KeyType, entry.KeyData)
		} else {
			line = fmt.Sprintf("%s %s %s\n", entry.HostPattern, entry.KeyType, entry.KeyData)
		}
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}
