package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/hookinput"
	"github.com/jryio/agentlink/internal/link"
	"github.com/jryio/agentlink/internal/safefs"
)

func (a *application) runCheck(ctx context.Context, args []string) error {
	flags := newFlagSet("check")
	var pairs stringList
	flags.Var(&pairs, "pair", "check only this pair (repeatable)")
	if err := flags.Parse(args); err != nil {
		return a.flagError("check", err)
	}
	if flags.NArg() != 0 {
		return a.usageError("check does not accept positional arguments")
	}
	return a.withEngine(func(doc *config.Document, engine *link.Engine) error {
		selected, err := selectedPairs(&doc.Config, pairs, true)
		if err != nil {
			return a.usageError(err.Error())
		}
		report := engine.Check(ctx, selected)
		if err := ctx.Err(); err != nil {
			return err
		}
		if !a.global.quiet {
			if err := a.printReport(report); err != nil {
				return err
			}
		}
		if !report.Clean() {
			return &ExitError{Code: exitDrift}
		}
		return nil
	})
}

func (a *application) runSync(ctx context.Context, args []string) error {
	flags := newFlagSet("sync")
	from := flags.String("from", "", "peer to copy from: claude or codex")
	apply := flags.Bool("apply", false, "apply the displayed plan")
	prune := flags.Bool("prune", false, "delete target-only files (requires --apply to mutate)")
	var pairs stringList
	flags.Var(&pairs, "pair", "sync only this pair (repeatable)")
	if err := flags.Parse(args); err != nil {
		return a.flagError("sync", err)
	}
	if flags.NArg() != 0 {
		return a.usageError("sync does not accept positional arguments")
	}
	if *from != string(link.SideClaude) && *from != string(link.SideCodex) {
		return a.usageError("sync requires --from claude or --from codex")
	}
	return a.withEngine(func(doc *config.Document, engine *link.Engine) error {
		selected, err := selectedPairs(&doc.Config, pairs, false)
		if err != nil {
			return a.usageError(err.Error())
		}
		plan, err := engine.PlanSync(ctx, link.Side(*from), *prune, selected)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		deferJSONOutput := a.global.format == "json" && *apply && len(plan.Operations) > 0 && len(plan.Unresolved) == 0
		if !a.global.quiet && !deferJSONOutput {
			if err := a.printPlan(plan, *apply); err != nil {
				return err
			}
		}
		if len(plan.Unresolved) > 0 {
			return &ExitError{Code: exitDrift, Err: errors.New("sync has unresolved findings; narrow --pair or resolve them manually")}
		}
		if !*apply || len(plan.Operations) == 0 {
			return nil
		}
		if err := engine.Apply(ctx, plan); err != nil {
			return err
		}
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
		if !a.global.quiet {
			if a.global.format == "json" {
				if err := a.printSyncResult(plan, report); err != nil {
					return err
				}
			} else {
				output := printer{writer: a.streams.Out}
				output.println()
				if output.err != nil {
					return output.err
				}
				if err := a.printReport(report); err != nil {
					return err
				}
			}
		}
		if !report.Clean() {
			return &ExitError{Code: exitDrift, Err: errors.New("post-sync verification still found drift")}
		}
		return nil
	})
}

func (a *application) runGuard(ctx context.Context, args []string, reminder bool) error {
	name := "guard"
	if reminder {
		name = "remind"
	}
	flags := newFlagSet(name)
	agent := flags.String("agent", "human", "output adapter: human, claude, or codex")
	if err := flags.Parse(args); err != nil {
		return a.flagError(name, err)
	}
	if *agent != "human" && *agent != "claude" && *agent != "codex" {
		return a.usageError("--agent must be human, claude, or codex")
	}
	changed := flags.Args()
	if len(changed) == 0 {
		parsed, err := hookinput.Parse(ctx, a.streams.In)
		if err != nil {
			return fmt.Errorf("read changed paths: %w", err)
		}
		changed = parsed
	}
	return a.withEngine(func(_ *config.Document, engine *link.Engine) error {
		violations, err := engine.Guard(ctx, changed)
		if err != nil {
			return err
		}
		if err := a.printGuard(violations, *agent, reminder); err != nil {
			return err
		}
		if !reminder && len(violations) > 0 {
			return &ExitError{Code: exitDrift}
		}
		return nil
	})
}

func (a *application) runList(args []string) error {
	if len(args) != 0 {
		return a.usageError("list does not accept arguments")
	}
	doc, err := a.loadDocument()
	if err != nil {
		return err
	}
	if a.global.quiet {
		return nil
	}
	if a.global.format == "json" {
		return writeJSON(a.streams.Out, struct {
			Config      string              `json:"config"`
			Roots       map[string]string   `json:"roots"`
			Pairs       []config.Pair       `json:"pairs"`
			MCPServers  []config.MCPServer  `json:"mcp_servers,omitempty"`
			Activations []config.Activation `json:"activations,omitempty"`
			Exceptions  []config.Exception  `json:"exceptions,omitempty"`
		}{doc.Path, doc.Roots, doc.Config.Pairs, doc.Config.MCPServers, doc.Config.Activations, doc.Config.Exceptions})
	}
	output := printer{writer: a.streams.Out}
	output.printf("Config  %s\n\nSources\n", doc.Path)
	names := make([]string, 0, len(doc.Roots))
	for name := range doc.Roots {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		optional := ""
		if doc.Config.Sources[name].Optional {
			optional = " (optional)"
		}
		output.printf("  %-18s %s%s\n", name, doc.Roots[name], optional)
	}
	output.println("\nPairs")
	for _, pair := range doc.Config.Pairs {
		output.printf("  %-18s %-4s %s:%s ↔ %s:%s\n", pair.ID, pair.Kind, pair.Claude.Source, pair.Claude.Path, pair.Codex.Source, pair.Codex.Path)
	}
	if len(doc.Config.MCPServers) > 0 {
		output.println("\nMCP wiring")
		for _, server := range doc.Config.MCPServers {
			output.printf("  %-18s %s:%s ↔ %s:%s\n", server.ID, server.Claude.Config.Path, server.Claude.Server, server.Codex.Config.Path, server.Codex.Server)
		}
	}
	if len(doc.Config.Activations) > 0 {
		output.println("\nLive activations")
		for _, activation := range doc.Config.Activations {
			output.printf("  %-18s %s:%s ← %s:%s\n", activation.ID, activation.Live.Source, activation.Live.Path, activation.Expected.Source, activation.Expected.Path)
		}
	}
	if len(doc.Config.Exceptions) > 0 {
		output.println("\nIntentional divergences")
		for _, exception := range doc.Config.Exceptions {
			output.printf("  %s %s — %s\n", exception.Pair, strings.Join(exception.Paths, ", "), exception.Reason)
		}
	}
	return output.err
}

func (a *application) runValidate(args []string) error {
	if len(args) != 0 {
		return a.usageError("validate does not accept arguments")
	}
	doc, err := a.loadDocument()
	if err != nil {
		return err
	}
	if a.global.quiet {
		return nil
	}
	if a.global.format == "json" {
		return writeJSON(a.streams.Out, map[string]any{"config": doc.Path, "valid": true})
	}
	output := printer{writer: a.streams.Out}
	output.printf("✓ valid %s\n", doc.Path)
	return output.err
}

func (a *application) runDoctor(args []string) (err error) {
	if len(args) != 0 {
		return a.usageError("doctor does not accept arguments")
	}
	doc, err := a.loadDocument()
	if err != nil {
		return err
	}
	roots, err := safefs.Open(doc)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, roots.Close()) }()
	if a.global.quiet {
		return nil
	}
	if a.global.format == "json" {
		available := make(map[string]bool, len(doc.Roots))
		for name := range doc.Roots {
			available[name] = roots.Available(name)
		}
		return writeJSON(a.streams.Out, map[string]any{
			"config": doc.Path, "valid": true, "sources": available,
		})
	}
	output := printer{writer: a.streams.Out}
	output.printf("✓ config  %s\n", doc.Path)
	names := mapsKeys(doc.Roots)
	slices.Sort(names)
	for _, name := range names {
		mark := "✓"
		if !roots.Available(name) {
			mark = "○"
		}
		output.printf("%s source  %-18s %s\n", mark, name, doc.Roots[name])
	}
	output.printf("✓ schema  version %d\n", config.CurrentVersion)
	return output.err
}

func (a *application) runSchema(args []string) error {
	if len(args) != 0 {
		return a.usageError("schema does not accept arguments; redirect stdout to a file")
	}
	_, err := a.streams.Out.Write(config.Schema())
	return err
}

func (a *application) runInit(args []string) error {
	flags := newFlagSet("init")
	force := flags.Bool("force", false, "replace existing generated files")
	if err := flags.Parse(args); err != nil {
		return a.flagError("init", err)
	}
	if flags.NArg() > 1 {
		return a.usageError("init accepts at most one config path")
	}
	configPath := a.global.config
	if flags.NArg() == 1 {
		if configPath != "" {
			return a.usageError("choose either --config or an init path, not both")
		}
		configPath = flags.Arg(0)
	}
	if configPath == "" {
		configPath = "agentlink.yaml"
	}
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(a.streams.CWD, configPath)
	}
	configPath = filepath.Clean(configPath)
	schemaPath := filepath.Join(filepath.Dir(configPath), "agentlink.schema.json")
	if configPath == schemaPath {
		return a.usageError("config path collides with generated schema path")
	}
	for _, generatedPath := range []string{configPath, schemaPath} {
		if err := checkGeneratedWritable(generatedPath, *force); err != nil {
			return err
		}
	}
	if err := writeGenerated(configPath, config.Sample(), *force); err != nil {
		return err
	}
	if err := writeGenerated(schemaPath, config.Schema(), *force); err != nil {
		return fmt.Errorf("write editor schema: %w", err)
	}
	if a.global.quiet {
		return nil
	}
	if a.global.format == "json" {
		return writeJSON(a.streams.Out, map[string]string{"config": configPath, "schema": schemaPath})
	}
	output := printer{writer: a.streams.Out}
	output.printf("✓ created %s\n✓ created %s\n\nNext: edit the pairs, then run `agentlink check`.\n", configPath, schemaPath)
	return output.err
}

func (a *application) withEngine(fn func(*config.Document, *link.Engine) error) (err error) {
	doc, err := a.loadDocument()
	if err != nil {
		return err
	}
	roots, err := safefs.Open(doc)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, roots.Close()) }()
	engine, err := link.New(doc, roots)
	if err != nil {
		return err
	}
	return fn(doc, engine)
}

func (a *application) loadDocument() (*config.Document, error) {
	path, err := config.Find(a.global.config, a.streams.CWD)
	if err != nil {
		return nil, err
	}
	return config.Load(path, a.streams.CWD)
}
