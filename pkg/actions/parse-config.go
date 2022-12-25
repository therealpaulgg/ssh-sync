package actions

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

type Host struct {
	Host   string            `json:"host"`
	Values map[string]string `json:"values"`
}

func parseSshConfig() ([]Host, error) {
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
	var hosts []Host
	scanner := bufio.NewScanner(file)
	var currentHost *Host
	re := regexp.MustCompile(`^\s+(\w+)[ =](.+)$`)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Host ") {
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &Host{
				Host:   strings.TrimPrefix(line, "Host "),
				Values: make(map[string]string),
			}
		} else if re.Match([]byte(line)) {
			currentHost.Values[re.FindStringSubmatch(line)[1]] = re.FindStringSubmatch(line)[2]
		}
	}
	if currentHost != nil {
		hosts = append(hosts, *currentHost)
	}
	return hosts, nil
}

func ParseConfig(c *cli.Context) error {
	hosts, err := parseSshConfig()
	if err != nil {
		return err
	}
	for _, host := range hosts {
		fmt.Println(host)
	}
	return nil
}
