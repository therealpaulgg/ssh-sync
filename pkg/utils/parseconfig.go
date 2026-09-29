package utils

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/kevinburke/ssh_config"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func ParseConfig() ([]models.Host, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(homeDir, ".ssh", "config")
	return parseConfigFile(configPath, homeDir, map[string]bool{})
}

func parseConfigFile(path, homeDir string, visited map[string]bool) ([]models.Host, error) {
	if visited[path] {
		return nil, nil
	}
	visited[path] = true

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := ssh_config.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	hosts := collectHosts(cfg, homeDir)

	includePaths, err := resolveIncludes(content, path, homeDir)
	if err != nil {
		return nil, err
	}
	for _, includePath := range includePaths {
		includeHosts, err := parseConfigFile(includePath, homeDir, visited)
		if err != nil {
			return nil, err
		}
		if len(includeHosts) > 0 {
			hosts = append(hosts, includeHosts...)
		}
	}

	return hosts, nil
}

func collectHosts(cfg *ssh_config.Config, homeDir string) []models.Host {
	var hosts []models.Host

	for _, h := range cfg.Hosts {
		values := map[string][]string{}
		var identityFiles []string

		for _, node := range h.Nodes {
			kv, ok := node.(*ssh_config.KV)
			if !ok {
				continue
			}
			key := kv.Key
			value := strings.TrimSpace(strings.Trim(kv.Value, `"`))

			if strings.EqualFold(key, "identityfile") {
				normalized := normalizeIdentityFilePath(value, homeDir)
				if normalized != "" {
					identityFiles = append(identityFiles, normalized)
				}
				continue
			}

			values[key] = append(values[key], value)
		}

		if len(values) == 0 && len(identityFiles) == 0 {
			continue
		}

		for _, pattern := range h.Patterns {
			hosts = append(hosts, models.Host{
				Host:          pattern.String(),
				IdentityFiles: identityFiles,
				Values:        cloneValues(values),
			})
		}
	}

	return hosts
}

func cloneValues(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string][]string, len(values))
	for key, vals := range values {
		copyVals := make([]string, len(vals))
		copy(copyVals, vals)
		cloned[key] = copyVals
	}

	return cloned
}

func normalizeIdentityFilePath(value, homeDir string) string {
	if value == "" {
		return ""
	}

	cleanHome := filepath.Clean(homeDir)
	identityFile := strings.TrimSpace(value)
	identityFile = strings.Trim(identityFile, `"`)

	if strings.HasPrefix(identityFile, "~") {
		identityFile = filepath.Join(cleanHome, strings.TrimPrefix(identityFile, "~"))
	}

	if runtime.GOOS == "windows" {
		lowerValue := strings.ToLower(identityFile)
		lowerHome := strings.ToLower(cleanHome)
		if strings.HasPrefix(lowerValue, lowerHome) {
			identityFile = identityFile[len(cleanHome):]
		}
	} else if strings.HasPrefix(identityFile, cleanHome) {
		identityFile = strings.TrimPrefix(identityFile, cleanHome)
	}

	identityFile = strings.TrimPrefix(identityFile, string(filepath.Separator))

	return filepath.ToSlash(identityFile)
}

func resolveIncludes(content []byte, basePath, homeDir string) ([]string, error) {
	var includes []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	baseDir := filepath.Dir(basePath)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := splitDirective(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "include") {
			continue
		}

		for _, pattern := range fields[1:] {
			resolved := pattern
			if strings.HasPrefix(pattern, "~") {
				resolved = filepath.Join(homeDir, strings.TrimPrefix(pattern, "~"))
			} else if !filepath.IsAbs(pattern) {
				resolved = filepath.Join(baseDir, pattern)
			}

			matches, err := filepath.Glob(resolved)
			if err != nil {
				return nil, err
			}
			includes = append(includes, matches...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return includes, nil
}

func splitDirective(line string) []string {
	return strings.FieldsFunc(line, func(r rune) bool {
		return unicode.IsSpace(r) || r == '='
	})
}
