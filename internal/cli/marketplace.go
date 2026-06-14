package cli

import (
	"fmt"
	"os"
	"strings"

	"reasonix/internal/marketplace"
)

func marketplaceCommand(args []string) int {
	reg := marketplace.DefaultRegistry()
	if len(args) == 0 {
		fmt.Println("reasonix marketplace — community skill registry (agentskills.io-compatible)")
		fmt.Println()
		fmt.Printf("  %d skills available. Commands:\n", reg.Len())
		fmt.Println("  reasonix marketplace list              — list all skills")
		fmt.Println("  reasonix marketplace search <query>    — search by name/description/tag")
		fmt.Println("  reasonix marketplace tags              — list all tags")
		fmt.Println("  reasonix marketplace tag <tag>         — filter by tag")
		fmt.Println("  reasonix marketplace install <name>    — show install instructions")
		fmt.Println("  reasonix marketplace sync              — sync skills from LobeHub marketplace")
		fmt.Println()
		fmt.Println("  Skills are installed via: reasonix install-source install --source <url>")
		return 0
	}

	switch args[0] {
	case "list":
		entries := reg.List()
		for _, e := range entries {
			fmt.Printf("  %-35s ★%.1f  %s\n", e.Name, e.Rating, strings.Join(e.Tags, ", "))
		}
		fmt.Printf("\n%d skills\n", len(entries))
	case "search":
		query := strings.Join(args[1:], " ")
		if query == "" {
			fmt.Fprintln(os.Stderr, "usage: reasonix marketplace search <query>")
			return 2
		}
		results := reg.Search(query)
		if len(results) == 0 {
			fmt.Printf("no skills match %q\n", query)
			return 0
		}
		for _, e := range results {
			fmt.Printf("  %-35s ★%.1f\n    %s\n    tags: %s\n\n", e.Name, e.Rating, e.Description, strings.Join(e.Tags, ", "))
		}
		fmt.Printf("%d skills match %q\n", len(results), query)
	case "tags":
		for _, t := range reg.Tags() {
			fmt.Println(" ", t)
		}
	case "tag":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: reasonix marketplace tag <tag>")
			return 2
		}
		results := reg.ByTag(args[1])
		if len(results) == 0 {
			fmt.Printf("no skills with tag %q\n", args[1])
			return 0
		}
		for _, e := range results {
			fmt.Printf("  %-35s ★%.1f  %s\n", e.Name, e.Rating, e.Description)
		}
	case "install":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: reasonix marketplace install <skill-name>")
			return 2
		}
		e := reg.ByName(args[1])
		if e == nil {
			fmt.Fprintf(os.Stderr, "skill %q not found in registry\n", args[1])
			return 1
		}
		fmt.Printf("Skill: %s (★%.1f)\n", e.Name, e.Rating)
		fmt.Printf("Author: %s\n", e.Author)
		fmt.Printf("Description: %s\n", e.Description)
		fmt.Println()
		fmt.Printf("To install, run:\n")
		fmt.Printf("  reasonix install-source install --source %s\n", e.URL)
	case "sync":
		client := marketplace.NewLobeHubClient("", "")
		host, _ := os.Hostname()
		cid, csec, err := client.Register("reasonix-hermes", "cli", "reasonix-cli-"+host)
		if err != nil {
			fmt.Fprintf(os.Stderr, "lobehub register: %v\n", err)
			return 1
		}
		fmt.Printf("Registered client: %s\n", cid)
		fmt.Printf("Syncing from LobeHub marketplace...\n")
		fetched, added, err := reg.SyncFromLobeHub(client, "", "installCount", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "lobehub sync: %v\n", err)
			return 1
		}
		fmt.Printf("Fetched %d skills, added %d new to registry (total: %d)\n", fetched, added, reg.Len())
		fmt.Printf("Credentials saved in memory (client_id=%s). Set in reasonix.toml [marketplace.lobehub] to persist.\n", cid)
		fmt.Printf("  client_secret = %s\n", csec)
	default:
		fmt.Fprintf(os.Stderr, "unknown marketplace command: %s\n", args[0])
		return 2
	}
	return 0
}
