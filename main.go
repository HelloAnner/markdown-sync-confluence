package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/markdown"
)

const (
	helpEpilog = `
Configuration Priorities (highest to lowest):
  1. Command line arguments:
     md2kms test.md --url https://your-domain.atlassian.net --username your.email@domain.com --password your-token --space SPACEKEY

  2. Environment variables:
     export KMS_URL=https://your-domain.atlassian.net
     export KMS_USERNAME=your.email@domain.com
     export KMS_PASSWORD=your-api-token
     export KMS_SPACE=SPACEKEY
     md2kms test.md

  3. Configuration file:
     md2kms test.md --config config.yml

Examples:
  # Using command line arguments
  md2kms test.md --url https://your-domain.atlassian.net --username your.email@domain.com --password your-token --space SPACEKEY --parent 123456

  # Using file name as page title
  md2kms test.md --parent 123456

  # Specifying page title
  md2kms test.md --title "My Document" --parent 123456
`
)

func main() {
	// Define command line flags
	markdownFile := flag.String("file", "", "Path to the markdown file to publish")
	titleFlag := flag.String("title", "", "Confluence page title (defaults to file name)")
	parentFlag := flag.String("parent", "", "Parent page ID")
	configFlag := flag.String("config", "", "Path to config file")
	
	// Confluence configuration flags
	urlFlag := flag.String("url", "", "Confluence URL (e.g. https://your-domain.atlassian.net)")
	usernameFlag := flag.String("username", "", "Confluence username/email")
	passwordFlag := flag.String("password", "", "Confluence API Token")
	spaceFlag := flag.String("space", "", "Confluence Space Key")
	
	// Add aliases for flags
	flag.StringVar(titleFlag, "t", "", "Short for --title")
	flag.StringVar(parentFlag, "p", "", "Short for --parent")
	flag.StringVar(configFlag, "c", "", "Short for --config")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] markdown_file\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, helpEpilog)
	}

	flag.Parse()

	// Get positional arguments
	args := flag.Args()
	if len(args) > 0 {
		*markdownFile = args[0]
	}

	if *markdownFile == "" {
		fmt.Println("❌ Error: Markdown file path is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create CLI config map
	cliConfig := map[string]string{
		"url":      *urlFlag,
		"username": *usernameFlag,
		"password": *passwordFlag,
		"space":    *spaceFlag,
	}

	// Load configuration with priority handling
	cfg, err := config.LoadConfig(*configFlag, cliConfig)
	if err != nil {
		fmt.Printf("❌ Error: %s\n", err)
		os.Exit(1)
	}

	// Create markdown-to-confluence converter
	converter := markdown.NewConverter(cfg)

	// Get title from flag or filename
	title := *titleFlag
	if title == "" {
		title = filepath.Base(*markdownFile)
		extension := filepath.Ext(title)
		title = title[0 : len(title)-len(extension)]
	}

	// Publish markdown to confluence
	err = converter.Publish(*markdownFile, title, *parentFlag)
	if err != nil {
		fmt.Printf("❌ Error: %s\n", err)
		os.Exit(1)
	}
}
