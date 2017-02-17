package main

import (
	"fmt"
	"log"
	"sftp"
)

func textReportOut() {
	var pubList, err = sftp.BuildPublishers(SFTPPath)
	if err != nil {
		log.Fatalf("Error: Unable to read publisher directories: %s\n", SFTPPath, err)
	}

	for _, pub := range pubList {
		if len(pub.Issues) == 0 {
			continue
		}

		fmt.Println("Publisher:", pub.Name)

		for _, issue := range pub.Issues {
			fmt.Printf("  Issue: %s", issue.RelativePath)
			if issue.Errors.Len() != 0 {
				fmt.Printf("    *** Error: %s\n", issue.Errors)
				continue
			}
			fmt.Println()

			for _, pdf := range issue.PDFs {
				fmt.Printf("    PDF: %s", pdf.RelativePath)
				if pdf.Errors.Len() != 0 {
					fmt.Printf("    *** Error: %s\n", pdf.Errors)
					continue
				}
				fmt.Println()
			}
		}
	}
}
