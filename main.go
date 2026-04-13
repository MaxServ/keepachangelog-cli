package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	changelog "github.com/xmidt-org/gokeepachangelog"
	"github.com/urfave/cli/v3"
)

const defaultFile = "CHANGELOG.md"

func main() {
	cmd := &cli.Command{
		Name:    "changelog",
		Usage:   "Manage your CHANGELOG.md (Keep a Changelog format)",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Value:   defaultFile,
				Usage:   "path to CHANGELOG.md",
			},
		},
		Commands: []*cli.Command{
			initCmd(),
			addCmd(),
			releaseCmd(),
			showCmd(),
			yankCmd(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// --- init ---

func initCmd() *cli.Command {
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "Create a new CHANGELOG.md",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "repo",
				Usage: "repository URL for comparison links (e.g. https://github.com/org/repo)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			path := cmd.String("file")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists", path)
			}

			cl := &changelog.Changelog{
				Title: "Changelog",
				Description: []string{
					"All notable changes to this project will be documented in this file.",
					"",
					"The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),",
					"and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).",
				},
				Releases: []changelog.Release{
					{Version: "Unreleased"},
				},
			}

			repo := cmd.String("repo")
			if repo != "" {
				repo = strings.TrimRight(repo, "/")
				cl.Links = []changelog.Link{
					{Version: "Unreleased", Url: repo + "/compare/HEAD"},
				}
			}

			return writeChangelog(path, cl)
		},
	}
}

// --- add ---

func addCmd() *cli.Command {
	return &cli.Command{
		Name:    "add",
		Aliases: []string{"a"},
		Usage:   "Add an entry to the Unreleased section",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "type",
				Aliases:  []string{"t"},
				Usage:    "change type: added, changed, deprecated, removed, fixed, security",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "message",
				Aliases:  []string{"m"},
				Usage:    "changelog entry message",
				Required: true,
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			path := cmd.String("file")
			changeType := strings.ToLower(cmd.String("type"))
			message := cmd.String("message")

			cl, err := readChangelog(path)
			if err != nil {
				return err
			}

			idx := findRelease(cl, "Unreleased")
			if idx < 0 {
				cl.Releases = append([]changelog.Release{{Version: "Unreleased"}}, cl.Releases...)
				idx = 0
			}

			entry := "- " + message
			rel := &cl.Releases[idx]

			switch changeType {
			case "added":
				rel.Added = append(rel.Added, entry)
			case "changed":
				rel.Changed = append(rel.Changed, entry)
			case "deprecated":
				rel.Deprecated = append(rel.Deprecated, entry)
			case "removed":
				rel.Removed = append(rel.Removed, entry)
			case "fixed":
				rel.Fixed = append(rel.Fixed, entry)
			case "security":
				rel.Security = append(rel.Security, entry)
			default:
				return fmt.Errorf("unknown type %q (use: added, changed, deprecated, removed, fixed, security)", changeType)
			}

			if err := writeChangelog(path, cl); err != nil {
				return err
			}

			fmt.Printf("Added [%s]: %s\n", changeType, message)
			return nil
		},
	}
}

// --- release ---

func releaseCmd() *cli.Command {
	return &cli.Command{
		Name:    "release",
		Aliases: []string{"r"},
		Usage:   "Promote Unreleased to a new version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Aliases:  []string{"v"},
				Usage:    "version number (e.g. 1.0.0)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "release date in YYYY-MM-DD format (default: today)",
			},
			&cli.StringFlag{
				Name:  "repo",
				Usage: "repository URL for comparison links",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			path := cmd.String("file")
			version := cmd.String("version")
			dateStr := cmd.String("date")
			repo := cmd.String("repo")

			cl, err := readChangelog(path)
			if err != nil {
				return err
			}

			idx := findRelease(cl, "Unreleased")
			if idx < 0 {
				return fmt.Errorf("no Unreleased section found")
			}

			unreleased := cl.Releases[idx]
			if isEmpty(unreleased) {
				return fmt.Errorf("Unreleased section is empty, nothing to release")
			}

			var date time.Time
			if dateStr != "" {
				date, err = time.Parse("2006-01-02", dateStr)
				if err != nil {
					return fmt.Errorf("invalid date %q (use YYYY-MM-DD)", dateStr)
				}
			} else {
				date = time.Now()
			}

			// Create the new release from unreleased content
			newRelease := changelog.Release{
				Version:    version,
				Date:       &date,
				Added:      unreleased.Added,
				Changed:    unreleased.Changed,
				Deprecated: unreleased.Deprecated,
				Removed:    unreleased.Removed,
				Fixed:      unreleased.Fixed,
				Security:   unreleased.Security,
			}

			// Reset Unreleased
			cl.Releases[idx] = changelog.Release{Version: "Unreleased"}

			// Insert new release after Unreleased
			releases := make([]changelog.Release, 0, len(cl.Releases)+1)
			releases = append(releases, cl.Releases[:idx+1]...)
			releases = append(releases, newRelease)
			releases = append(releases, cl.Releases[idx+1:]...)
			cl.Releases = releases

			// Update links if repo URL is available
			if repo != "" {
				repo = strings.TrimRight(repo, "/")
				cl.Links = updateLinks(cl, repo)
			}

			if err := writeChangelog(path, cl); err != nil {
				return err
			}

			fmt.Printf("Released %s (%s)\n", version, date.Format("2006-01-02"))
			return nil
		},
	}
}

// --- show ---

func showCmd() *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"s"},
		Usage:   "Show the changelog or a specific version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "show a specific version (default: show all)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			path := cmd.String("file")
			version := cmd.String("version")

			cl, err := readChangelog(path)
			if err != nil {
				return err
			}

			if version == "" {
				fmt.Print(collapseBlankLines(cl.ToMarkdown()))
				return nil
			}

			idx := findRelease(cl, version)
			if idx < 0 {
				return fmt.Errorf("version %q not found", version)
			}

			fmt.Print(collapseBlankLines(cl.Releases[idx].ToMarkdown()))
			return nil
		},
	}
}

// --- yank ---

func yankCmd() *cli.Command {
	return &cli.Command{
		Name:  "yank",
		Usage: "Mark a release as yanked",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Aliases:  []string{"v"},
				Usage:    "version to yank",
				Required: true,
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			path := cmd.String("file")
			version := cmd.String("version")

			cl, err := readChangelog(path)
			if err != nil {
				return err
			}

			idx := findRelease(cl, version)
			if idx < 0 {
				return fmt.Errorf("version %q not found", version)
			}

			cl.Releases[idx].Yanked = true

			if err := writeChangelog(path, cl); err != nil {
				return err
			}

			fmt.Printf("Marked %s as [YANKED]\n", version)
			return nil
		},
	}
}

// --- helpers ---

func readChangelog(path string) (*changelog.Changelog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	cl, err := changelog.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return cl, nil
}

func writeChangelog(path string, cl *changelog.Changelog) error {
	output := collapseBlankLines(cl.ToMarkdown())
	return os.WriteFile(path, []byte(output), 0644)
}

// collapseBlankLines reduces runs of 3+ consecutive blank lines to 2.
func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blanks := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks > 2 {
				continue
			}
		} else {
			blanks = 0
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func findRelease(cl *changelog.Changelog, version string) int {
	for i := range cl.Releases {
		if strings.EqualFold(cl.Releases[i].Version, version) {
			return i
		}
	}
	return -1
}

func isEmpty(r changelog.Release) bool {
	return len(r.Added) == 0 &&
		len(r.Changed) == 0 &&
		len(r.Deprecated) == 0 &&
		len(r.Removed) == 0 &&
		len(r.Fixed) == 0 &&
		len(r.Security) == 0
}

func updateLinks(cl *changelog.Changelog, repo string) []changelog.Link {
	var links []changelog.Link

	for i, rel := range cl.Releases {
		if rel.Version == "Unreleased" {
			// Unreleased compares against the next version
			if i+1 < len(cl.Releases) {
				links = append(links, changelog.Link{
					Version: "Unreleased",
					Url:     fmt.Sprintf("%s/compare/%s...HEAD", repo, cl.Releases[i+1].Version),
				})
			}
			continue
		}

		if i+1 < len(cl.Releases) {
			// Find the next older version (skip Unreleased)
			nextIdx := i + 1
			for nextIdx < len(cl.Releases) && cl.Releases[nextIdx].Version == "Unreleased" {
				nextIdx++
			}
			if nextIdx < len(cl.Releases) {
				links = append(links, changelog.Link{
					Version: rel.Version,
					Url:     fmt.Sprintf("%s/compare/%s...%s", repo, cl.Releases[nextIdx].Version, rel.Version),
				})
			} else {
				links = append(links, changelog.Link{
					Version: rel.Version,
					Url:     fmt.Sprintf("%s/releases/tag/%s", repo, rel.Version),
				})
			}
		} else {
			// Oldest release - link to tag
			links = append(links, changelog.Link{
				Version: rel.Version,
				Url:     fmt.Sprintf("%s/releases/tag/%s", repo, rel.Version),
			})
		}
	}

	return links
}
