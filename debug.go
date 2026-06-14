package main

import "log"

// Debug controls verbose logging of hot-path events (per-request file event
// registration, command processing, read/write byte counts). Enabled via
// environment variable GODIS_DEBUG=1 or by setting the flag directly.
// When false (default, production), only errors, warnings, and lifecycle
// events (accept, close, startup) are logged.
var Debug bool

func init() {
	// Reserved for future env-var parsing.
	// Set Debug = true in tests or during development to re-enable verbose logs.
}

func debugf(format string, args ...interface{}) {
	if Debug {
		log.Printf(format, args...)
	}
}
