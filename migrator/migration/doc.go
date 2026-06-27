// Package migration provides a small SQL migration runner for applications that
// own their database connection and migration folder.
//
// The package is intentionally framework-neutral: it does not load environment
// files, open database connections, or assume a specific project layout.
package migration
