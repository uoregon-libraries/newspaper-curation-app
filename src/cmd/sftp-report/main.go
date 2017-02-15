package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bashconf"
	"sftp"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
}

// SFTPPath gets the configured path to the SFTP root where each publisher
// directory resides
var SFTPPath string

// IsDir returns true if the given path exists and is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func getConf() {
	var p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	var c bashconf.Config
	c, err = bashconf.ReadFile(opts.ConfigFile)
	if err != nil {
		log.Fatal("Error parsing config file: %s", err)
	}

	SFTPPath = c["MASTER_PDF_UPLOAD_PATH"]
	if !IsDir(SFTPPath) {
		fmt.Fprintf(os.Stderr, "Error: Cannot access SFTP path %#v\n\n", SFTPPath)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}

func main() {
	getConf()
	var pubList, err = sftp.BuildPublishers(SFTPPath)
	if err != nil {
		log.Fatalf("Error: Unable to read publisher directories: %s\n", SFTPPath, err)
	}

	var i int
	for _, pub := range pubList {
		if len(pub.Issues) == 0 {
			continue
		}

		if i > 0 {
			fmt.Println()
		}
		i++
		fmt.Println("Publisher:", pub.Name)

		for _, issue := range pub.Issues {
			fmt.Printf("  Issue: %s", issue.RelPath)
			if issue.Error != nil {
				fmt.Printf("    *** Error: %s\n", issue.Error)
				continue
			}
			fmt.Println()

			for _, pdf := range issue.PDFs {
				fmt.Printf("    PDF: %s", filepath.Join(pub.Name, issue.Name, pdf.Name))
				if pdf.Error != nil {
					fmt.Printf("    *** Error: %s\n", pdf.Error)
					continue
				}
				fmt.Println()
			}
		}
	}
}
