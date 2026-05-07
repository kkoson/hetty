package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	// defaultAddr is the default address the HTTP proxy and admin interface listen on.
	defaultAddr = ":8080"
	// defaultAdminPath is the default path for the admin interface.
	defaultAdminPath = "/hetty/"
	// defaultDBPath is the default path for the database file.
	// Changed from "hetty.db" to store in a dedicated data directory.
	defaultDBPath = "data/hetty.db"
)

// version is set at build time using ldflags.
var version = "dev"

func main() {
	// Parse command-line flags.
	addr := flag.String("addr", defaultAddr, "Address to listen on (e.g. :8080)")
	adminPath := flag.String("adminPath", defaultAdminPath, "Path prefix for the admin interface")
	dbPath := flag.String("db", defaultDBPath, "Path to the database file")
	projName := flag.String("project", "", "Name of the project to open or create on startup")
	certFile := flag.String("cert", "", "Path to the CA certificate file (PEM format)")
	keyFile := flag.String("key", "", "Path to the CA private key file (PEM format)")
	printVersion := flag.Bool("version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: hetty [options]\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nVersion: %s\n", version)
	}

	flag.Parse()

	if *printVersion {
		fmt.Printf("hetty %s\n", version)
		os.Exit(0)
	}

	// Include date/time and file info in log output, but omit the date for cleaner logs.
	log.SetFlags(log.Ltime | log.Lshortfile)

	// Log startup configuration.
	log.Printf("[INFO] Starting hetty %s", version)
	log.Printf("[INFO] Listening on %s", *addr)
	log.Printf("[INFO] Admin interface path: %s", *adminPath)
	log.Printf("[INFO] Database path: %s", *dbPath)

	if *projName != "" {
		log.Printf("[INFO] Opening project: %s", *projName)
	}

	if *certFile != "" && *keyFile != "" {
		log.Printf("[INFO] Using CA certificate: %s", *certFile)
	} else if *certFile != "" || *keyFile != "" {
		log.Fatal("[ERROR] Both -cert and -key flags must be provided together")
	}

	// TODO: Initialize database, proxy, and HTTP server.
	// This will be wired up as the project grows.
	log.Fatal("[ERROR] Server initialization not yet implemented")
}
