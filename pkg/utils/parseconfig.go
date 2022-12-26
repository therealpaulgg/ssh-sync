package utils

import (
	"bufio"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func ParseConfig() ([]models.Host, error) {
	// parse the ssh config file and return a list of hosts
	// the ssh config file is located at ~/.ssh/config
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := path.Join(user.HomeDir, ".ssh", "config")
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	var hosts []models.Host
	scanner := bufio.NewScanner(file)
	var currentHost *models.Host
	re := regexp.MustCompile(`^\s+(\w+)[ =](.+)$`)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Host ") {
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &models.Host{
				Host:   strings.TrimPrefix(line, "Host "),
				Values: make(map[string]string),
			}
		} else if re.Match([]byte(line)) {
			key := re.FindStringSubmatch(line)[1]
			value := re.FindStringSubmatch(line)[2]
			if key == "IdentityFile" {
				identityFile := strings.TrimPrefix(value, user.HomeDir)
				currentHost.IdentityFile = identityFile
			} else {
				currentHost.Values[key] = value
			}

		}
	}
	if currentHost != nil {
		hosts = append(hosts, *currentHost)
	}
	return hosts, nil
}
