package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
	styles "github.com/charmbracelet/glamour/styles"
)

func PrintGenerate(generateMarkdown func(w io.Writer)) error {
	var b strings.Builder
	var w io.Writer = &b
	generateMarkdown(w)
	return Print(b.String())
}

func str(s string) *string {
	return &s
}

func Print(markdown string) error {
	width := 120
	// initialize glamour
	style := styles.NoTTYStyleConfig
	r, err := glamour.NewTermRenderer(
		// glamour.WithAutoStyle(),
		glamour.WithStyles(style),
		// glamour.WithColorProfile(lipgloss.ColorProfile()),
		// utils.GlamourStyle(style, isCode),
		glamour.WithWordWrap(int(width)), //nolint:gosec
		// glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return fmt.Errorf("unable to create renderer: %w", err)
	}
	out, err := r.Render(markdown)
	if err != nil {
		return fmt.Errorf("unable to render markdown: %w", err)
	}
	fmt.Print(out)
	return nil
}
