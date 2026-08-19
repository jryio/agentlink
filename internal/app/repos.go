package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/link"
)

// discoverRepos returns the immediate child directories of root that contain
// a .git entry (directory or worktree file), in lexical order.
func discoverRepos(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read repos root: %w", err)
	}
	repos := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		child := filepath.Join(root, entry.Name())
		if _, err := os.Lstat(filepath.Join(child, ".git")); err == nil {
			repos = append(repos, child)
		}
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories (directories containing .git) under %s", root)
	}
	return repos, nil
}

type repoCheckOutput struct {
	Repo   string        `json:"repo"`
	Name   string        `json:"name"`
	Report *reportOutput `json:"report,omitempty"`
	Error  string        `json:"error,omitempty"`
}

func (a *application) runCheckRepos(ctx context.Context, root string, pairs stringList) error {
	repos, err := a.repositories(root)
	if err != nil {
		return err
	}
	outputs := make([]repoCheckOutput, 0, len(repos))
	failed := 0
	clean := true
	for _, repo := range repos {
		output := repoCheckOutput{Repo: repo, Name: filepath.Base(repo)}
		var report link.Report
		err := a.withEngineAt(repo, func(doc *config.Document, engine *link.Engine) error {
			selected, err := selectedPairs(&doc.Config, pairs, true)
			if err != nil {
				return err
			}
			report = engine.Check(ctx, selected)
			return ctx.Err()
		})
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			output.Error = err.Error()
			failed++
			clean = false
		} else {
			reportOutput := newReportOutput(report)
			output.Report = &reportOutput
			if !report.Clean() {
				clean = false
			}
		}
		outputs = append(outputs, output)
	}
	if err := a.printCheckRepos(outputs); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d repositories failed", failed, len(repos))
	}
	if !clean {
		return &ExitError{Code: exitDrift}
	}
	return nil
}

func (a *application) printCheckRepos(outputs []repoCheckOutput) error {
	if a.global.quiet {
		return nil
	}
	if a.global.format == "json" {
		clean := true
		for _, output := range outputs {
			if output.Error != "" || output.Report == nil || !output.Report.Clean {
				clean = false
				break
			}
		}
		return writeJSON(a.streams.Out, struct {
			Repos []repoCheckOutput `json:"repos"`
			Clean bool              `json:"clean"`
		}{outputs, clean})
	}
	printer := printer{writer: a.streams.Out}
	for i, output := range outputs {
		printer.printf("=== %s  %s\n", output.Name, escapeTerminal(output.Repo))
		if output.Error != "" {
			printer.printf("error: %s\n", escapeTerminal(output.Error))
		} else if output.Report != nil {
			if err := a.printReport(link.Report{Pairs: output.Report.Pairs, Activations: output.Report.Activations}); err != nil {
				return err
			}
		}
		if i < len(outputs)-1 {
			printer.println()
		}
	}
	return printer.err
}

type repoSyncOutput struct {
	Repo         string        `json:"repo"`
	Name         string        `json:"name"`
	Plan         link.Plan     `json:"plan"`
	Applied      bool          `json:"applied"`
	Verification *reportOutput `json:"verification,omitempty"`
	Error        string        `json:"error,omitempty"`
}

func (a *application) runSyncRepos(ctx context.Context, root string, from link.Side, apply, prune bool, pairs stringList) error {
	repos, err := a.repositories(root)
	if err != nil {
		return err
	}
	outputs := make([]repoSyncOutput, 0, len(repos))
	failed := 0
	drifted := 0
	for _, repo := range repos {
		output := repoSyncOutput{Repo: repo, Name: filepath.Base(repo)}
		drift := false
		err := a.withEngineAt(repo, func(doc *config.Document, engine *link.Engine) error {
			selected, err := selectedPairs(&doc.Config, pairs, false)
			if err != nil {
				return err
			}
			plan, err := engine.PlanSync(ctx, from, prune, selected)
			if err != nil {
				return err
			}
			output.Plan = plan
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(plan.Unresolved) > 0 {
				drift = true
				return nil
			}
			if !apply || len(plan.Operations) == 0 {
				return nil
			}
			if err := engine.Apply(ctx, plan); err != nil {
				return err
			}
			output.Applied = true
			verification := selected
			if len(verification) == 0 {
				verification = make(map[string]bool, len(doc.Config.Pairs))
				for _, pair := range doc.Config.Pairs {
					verification[pair.ID] = true
				}
			}
			report := engine.Check(ctx, verification)
			if err := ctx.Err(); err != nil {
				return err
			}
			reportOutput := newReportOutput(report)
			output.Verification = &reportOutput
			if !report.Clean() {
				drift = true
			}
			return nil
		})
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			output.Error = err.Error()
			failed++
		} else if drift {
			drifted++
		}
		outputs = append(outputs, output)
	}
	if err := a.printSyncRepos(outputs, apply); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d repositories failed", failed, len(repos))
	}
	if drifted > 0 {
		return &ExitError{Code: exitDrift, Err: fmt.Errorf("%d of %d repositories still drift", drifted, len(repos))}
	}
	return nil
}

func (a *application) printSyncRepos(outputs []repoSyncOutput, apply bool) error {
	if a.global.quiet {
		return nil
	}
	if a.global.format == "json" {
		return writeJSON(a.streams.Out, struct {
			Repos []repoSyncOutput `json:"repos"`
		}{outputs})
	}
	printer := printer{writer: a.streams.Out}
	for i, output := range outputs {
		printer.printf("=== %s  %s\n", output.Name, escapeTerminal(output.Repo))
		if output.Error != "" {
			printer.printf("error: %s\n", escapeTerminal(output.Error))
		} else {
			if err := a.printPlan(output.Plan, apply); err != nil {
				return err
			}
			if output.Verification != nil {
				printer.println()
				if err := a.printReport(link.Report{Pairs: output.Verification.Pairs, Activations: output.Verification.Activations}); err != nil {
					return err
				}
			}
		}
		if i < len(outputs)-1 {
			printer.println()
		}
	}
	return printer.err
}

func (a *application) repositories(root string) ([]string, error) {
	if !filepath.IsAbs(root) {
		root = filepath.Join(a.streams.CWD, root)
	}
	return discoverRepos(filepath.Clean(root))
}
