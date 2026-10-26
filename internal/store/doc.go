// Package store is the only package that touches SQLite: schema, migrations,
// commits with history, trash, snapshots, JSON export and import, and locking
// (§7, §8, §9.2).
package store
