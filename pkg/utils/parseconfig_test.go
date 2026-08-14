package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func TestParseConfigCollectsGlobalAndHostValues(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0o700))

	config := `
GlobalKnownHostsFile /etc/ssh/known_hosts

Host foo bar
User alice
    Port 2222
    IdentityFile ~/.ssh/id_rsa

Host baz
	ProxyJump jump
`
	configPath := filepath.Join(sshDir, "config")
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

	hosts, err := parseConfigFile(configPath, home, map[string]bool{})
	require.NoError(t, err)

	requireHost := func(name string) models.Host {
		for _, h := range hosts {
			if h.Host == name {
				return h
			}
		}
		t.Fatalf("host %q not found", name)
		return models.Host{}
	}

	defaultHost := requireHost("*")
	assert.Equal(t, []string{"/etc/ssh/known_hosts"}, defaultHost.Values["GlobalKnownHostsFile"])

	foo := requireHost("foo")
	bar := requireHost("bar")
	for _, host := range []models.Host{foo, bar} {
		assert.Equal(t, []string{".ssh/id_rsa"}, host.IdentityFiles)
		assert.Equal(t, []string{"alice"}, host.Values["User"])
		assert.Equal(t, []string{"2222"}, host.Values["Port"])
	}

	baz := requireHost("baz")
	assert.Equal(t, []string{"jump"}, baz.Values["ProxyJump"])
}

func TestParseConfigResolvesIncludes(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0o700))

	configPath := filepath.Join(sshDir, "config")
	includePath := filepath.Join(sshDir, "extra.conf")

	config := `Include extra.conf

Host base
  User baseuser
`
	include := `
Host included
  IdentityFile ~/.ssh/id_included
`

	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))
	require.NoError(t, os.WriteFile(includePath, []byte(include), 0o600))

	hosts, err := parseConfigFile(configPath, home, map[string]bool{})
	require.NoError(t, err)

	requireHost := func(name string) models.Host {
		for _, h := range hosts {
			if h.Host == name {
				return h
			}
		}
		t.Fatalf("host %q not found", name)
		return models.Host{}
	}

	base := requireHost("base")
	assert.Equal(t, []string{"baseuser"}, base.Values["User"])

	included := requireHost("included")
	assert.Equal(t, []string{".ssh/id_included"}, included.IdentityFiles)
}
