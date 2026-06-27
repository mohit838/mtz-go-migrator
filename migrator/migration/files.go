package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func (r *Runner) loadFiles() ([]fileMigration, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	byVersion := make(map[string]*fileMigration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		direction := ""
		switch {
		case strings.HasSuffix(filename, ".up.sql"):
			direction = "up"
		case strings.HasSuffix(filename, ".down.sql"):
			direction = "down"
		default:
			continue
		}

		parts := strings.SplitN(strings.TrimSuffix(strings.TrimSuffix(filename, ".up.sql"), ".down.sql"), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", filename)
		}
		version, name := parts[0], parts[1]
		if !validVersion(version) || name == "" {
			return nil, fmt.Errorf("invalid migration filename: %s", filename)
		}
		item := byVersion[version]
		if item == nil {
			item = &fileMigration{version: version, name: name}
			byVersion[version] = item
		} else if item.name != name {
			return nil, fmt.Errorf("migration version %s has conflicting names: %s and %s", version, item.name, name)
		}
		path := filepath.Join(r.dir, filename)
		if direction == "up" {
			item.upPath = path
			checksum, err := checksumFile(path)
			if err != nil {
				return nil, err
			}
			item.checksum = checksum
		} else {
			item.downPath = path
		}
	}

	files := make([]fileMigration, 0, len(byVersion))
	for _, item := range byVersion {
		if item.upPath == "" || item.downPath == "" {
			return nil, fmt.Errorf("migration %s_%s must have up and down files", item.version, item.name)
		}
		files = append(files, *item)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})
	return files, nil
}

func validVersion(version string) bool {
	if len(version) != 14 {
		return false
	}
	for _, char := range version {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}
