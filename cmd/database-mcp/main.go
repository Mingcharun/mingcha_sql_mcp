package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Mingcharun/database-mcp/internal/service"
)

const applicationName = "Database MCP Server"

var (
	Version = "dev" // 通过 ldflags 在构建时注入实际版本号
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("%s v%s\n", applicationName, Version)
		fmt.Println("Integrated: MySQL, PostgreSQL, Redis, SQLite")
		return
	}

	app := service.New(applicationName, Version)

	log.Printf("Starting %s v%s...\n", applicationName, Version)
	if err := server.ServeStdio(app); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
