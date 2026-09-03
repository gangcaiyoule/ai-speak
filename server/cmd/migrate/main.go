// Package main provides the local database migration entry point.
package main

import "log"

// main is the migration command entry point. Database wiring is supplied by the next storage PR.
func main() {
	log.Println("database migration runner is ready; configure a PostgreSQL adapter before execution")
}
