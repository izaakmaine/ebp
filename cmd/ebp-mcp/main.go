package main

import (
	"flag"
	"log"

	"github.com/PithomLabs/ebp/internal/mcp"
	"github.com/PithomLabs/ebp/internal/service"
	"github.com/PithomLabs/ebp/internal/store"
)

func main() {
	dbPath := flag.String("db", "ebp.db", "SQLite database path")
	flag.Parse()

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ebp := service.New(db)
	s := mcp.NewServer(ebp)

	if err := s.ServeStdio(); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}
