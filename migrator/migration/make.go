package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (r *Runner) Make(name string) error {
	cleanName := sanitizeName(name)
	if cleanName == "" {
		return fmt.Errorf("migration name cannot be empty")
	}
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return err
	}

	version := r.now().UTC().Format("20060102150405")
	upPath := filepath.Join(r.dir, version+"_"+cleanName+".up.sql")
	downPath := filepath.Join(r.dir, version+"_"+cleanName+".down.sql")
	if err := writeNewFile(upPath, []byte("-- Write migration SQL here.\n")); err != nil {
		return err
	}
	if err := writeNewFile(downPath, []byte("-- Write rollback SQL here.\n")); err != nil {
		return err
	}

	r.println(r.logPrefix()+"Created", upPath)
	r.println(r.logPrefix()+"Created", downPath)
	return nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "_")
	return strings.Trim(name, "_")
}

func writeNewFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	return err
}
