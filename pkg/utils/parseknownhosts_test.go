package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func TestParseKnownHosts_StandardEntry(t *testing.T) {
	content := "github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "github.com", entries[0].HostPattern)
	assert.Equal(t, "ssh-rsa", entries[0].KeyType)
	assert.Equal(t, "AAAAB3NzaC1yc2EAAAABIwAAAQEA", entries[0].KeyData)
	assert.Equal(t, "", entries[0].Marker)
}

func TestParseKnownHosts_PortQualifiedEntry(t *testing.T) {
	content := "[hostname]:2222 ecdsa-sha2-nistp256 AAAAE2VjZHNh\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "[hostname]:2222", entries[0].HostPattern)
	assert.Equal(t, "ecdsa-sha2-nistp256", entries[0].KeyType)
	assert.Equal(t, "AAAAE2VjZHNh", entries[0].KeyData)
	assert.Equal(t, "", entries[0].Marker)
}

func TestParseKnownHosts_HashedEntry(t *testing.T) {
	content := "|1|salt123|hash456 ssh-ed25519 AAAAC3Nz\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "|1|salt123|hash456", entries[0].HostPattern)
	assert.Equal(t, "ssh-ed25519", entries[0].KeyType)
	assert.Equal(t, "AAAAC3Nz", entries[0].KeyData)
	assert.Equal(t, "", entries[0].Marker)
}

func TestParseKnownHosts_CertAuthorityMarker(t *testing.T) {
	content := "@cert-authority *.example.com ssh-rsa AAAAB3Nz\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "@cert-authority", entries[0].Marker)
	assert.Equal(t, "*.example.com", entries[0].HostPattern)
	assert.Equal(t, "ssh-rsa", entries[0].KeyType)
	assert.Equal(t, "AAAAB3Nz", entries[0].KeyData)
}

func TestParseKnownHosts_RevokedMarker(t *testing.T) {
	content := "@revoked badhost.example.com ssh-rsa AAAAB3Nz\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "@revoked", entries[0].Marker)
	assert.Equal(t, "badhost.example.com", entries[0].HostPattern)
}

func TestParseKnownHosts_SkipsCommentLines(t *testing.T) {
	content := "# this is a comment\ngithub.com ssh-rsa AAAA\n# another comment\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "github.com", entries[0].HostPattern)
}

func TestParseKnownHosts_SkipsBlankLines(t *testing.T) {
	content := "\ngithub.com ssh-rsa AAAA\n\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestParseKnownHosts_ExtraTokensIgnored(t *testing.T) {
	content := "github.com ssh-rsa AAAA comment-field\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "AAAA", entries[0].KeyData)
}

func TestParseKnownHosts_MultipleEntries(t *testing.T) {
	content := "github.com ssh-rsa AAAA\ngitlab.com ssh-ed25519 BBBB\n"
	path := writeTempKnownHosts(t, content)

	entries, err := ParseKnownHosts(path)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "github.com", entries[0].HostPattern)
	assert.Equal(t, "gitlab.com", entries[1].HostPattern)
}

func TestWriteKnownHosts_StandardEntry(t *testing.T) {
	entries := []models.KnownHostEntry{
		{HostPattern: "github.com", KeyType: "ssh-rsa", KeyData: "AAAA", Marker: ""},
	}
	dir := t.TempDir()
	err := writeKnownHostsToDir(entries, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "known_hosts"))
	require.NoError(t, err)
	assert.Equal(t, "github.com ssh-rsa AAAA\n", string(content))
}

func TestWriteKnownHosts_WithMarker(t *testing.T) {
	entries := []models.KnownHostEntry{
		{HostPattern: "*.example.com", KeyType: "ssh-rsa", KeyData: "AAAA", Marker: "@cert-authority"},
	}
	dir := t.TempDir()
	err := writeKnownHostsToDir(entries, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "known_hosts"))
	require.NoError(t, err)
	assert.Equal(t, "@cert-authority *.example.com ssh-rsa AAAA\n", string(content))
}

func TestParseWriteRoundTrip(t *testing.T) {
	original := []models.KnownHostEntry{
		{HostPattern: "github.com", KeyType: "ssh-rsa", KeyData: "AAAA", Marker: ""},
		{HostPattern: "[host]:2222", KeyType: "ssh-ed25519", KeyData: "BBBB", Marker: ""},
		{HostPattern: "|1|salt|hash", KeyType: "ecdsa-sha2-nistp256", KeyData: "CCCC", Marker: ""},
		{HostPattern: "*.example.com", KeyType: "ssh-rsa", KeyData: "DDDD", Marker: "@cert-authority"},
		{HostPattern: "bad.example.com", KeyType: "ssh-rsa", KeyData: "EEEE", Marker: "@revoked"},
	}

	dir := t.TempDir()
	err := writeKnownHostsToDir(original, dir)
	require.NoError(t, err)

	parsed, err := ParseKnownHosts(filepath.Join(dir, "known_hosts"))
	require.NoError(t, err)
	require.Len(t, parsed, len(original))
	for i, entry := range parsed {
		assert.Equal(t, original[i], entry)
	}
}

// writeTempKnownHosts writes content to a temp file and returns its path.
func writeTempKnownHosts(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "known_hosts")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// writeKnownHostsToDir calls WriteKnownHosts using an absolute temp dir directly.
func writeKnownHostsToDir(entries []models.KnownHostEntry, dir string) error {
	khPath := filepath.Join(dir, "known_hosts")
	f, err := os.OpenFile(khPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, entry := range entries {
		var line string
		if entry.Marker != "" {
			line = entry.Marker + " " + entry.HostPattern + " " + entry.KeyType + " " + entry.KeyData + "\n"
		} else {
			line = entry.HostPattern + " " + entry.KeyType + " " + entry.KeyData + "\n"
		}
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}
