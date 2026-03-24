package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func main() {
	lipgloss.SetColorProfile(termenv.NewOutput(os.Stdout).Profile)

	bucket := flag.String("bucket", "", "The S3 bucket to start in")
	region := flag.String("region", "us-east-1", "The AWS region (default: us-east-1)")
	profile := flag.String("profile", "default", "The AWS profile (default: default)")
	flag.Parse()

	m := model{
		profile:     *profile,
		region:      *region,
		startBucket: *bucket,
		state:       fileList,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	if fm, ok := finalModel.(model); ok && fm.s3Client != nil {
		fm.s3Client.close()
	}
}
