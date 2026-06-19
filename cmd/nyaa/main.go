package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/nyaa-tui/internal/opml"
	"github.com/user/nyaa-tui/internal/subscriptions"
	"github.com/user/nyaa-tui/internal/ui"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "import-opml" {
		os.Exit(runImportOPML(os.Args[2:]))
	}

	if err := ui.CheckDependencies(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		ui.InitialModel(),
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(), // hover to highlight, click to navigate
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// runImportOPML merges channels from an OPML export into the saved
// subscriptions. Returns a process exit code.
func runImportOPML(args []string) int {
	if len(args) < 1 {
		fmt.Println("usage: nyaa import-opml <file.opml> (>_<)")
		return 2
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Printf("couldn't read %s: %v (T_T)\n", args[0], err)
		return 1
	}

	parsed, err := opml.Parse(data)
	if err != nil {
		fmt.Printf("couldn't parse OPML: %v (＠_＠;)\n", err)
		return 1
	}

	subs, err := subscriptions.Load()
	if err != nil {
		fmt.Printf("couldn't load subscriptions: %v (T_T)\n", err)
		return 1
	}

	added := 0
	for _, s := range parsed {
		if s.ChannelID == "" {
			continue
		}
		if ok, _ := subs.Add(s.ChannelID, s.Name); ok {
			added++
		}
	}

	if err := subs.Save(); err != nil {
		fmt.Printf("couldn't save subscriptions: %v (T_T)\n", err)
		return 1
	}

	fmt.Printf("imported %d channel(s), %d new ♡ (now following %d) (≧◡≦)\n",
		len(parsed), added, subs.Count())
	return 0
}
