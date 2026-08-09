package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/link"
)

type reportOutput struct {
	Clean        bool                    `json:"clean"`
	FindingCount int                     `json:"finding_count"`
	Pairs        []link.PairReport       `json:"pairs"`
	Activations  []link.ActivationReport `json:"activations,omitempty"`
}

func newReportOutput(report link.Report) reportOutput {
	return reportOutput{
		Clean:        report.Clean(),
		FindingCount: report.FindingCount(),
		Pairs:        report.Pairs,
		Activations:  report.Activations,
	}
}

func (a *application) printReport(report link.Report) error {
	if a.global.format == "json" {
		return writeJSON(a.streams.Out, newReportOutput(report))
	}
	files := 0
	skipped := 0
	output := printer{writer: a.streams.Out}
	for _, pair := range report.Pairs {
		files += pair.Files
		if pair.Skipped {
			skipped++
			output.printf("○ %-20s skipped — %s\n", pair.ID, escapeTerminal(pair.Reason))
		}
		for _, finding := range pair.Findings {
			output.printf("× %-20s %-15s %s\n", pair.ID, finding.State, escapeTerminal(displayRelative(finding.Relative)))
			keys := make([]string, 0, len(finding.Paths))
			for id := range finding.Paths {
				keys = append(keys, id)
			}
			slices.Sort(keys)
			for _, id := range keys {
				output.printf("  %-8s %s\n", id, escapeTerminal(finding.Paths[id]))
			}
			if finding.Detail != "" {
				output.printf("  %s\n", escapeTerminal(finding.Detail))
			}
		}
	}
	for _, activation := range report.Activations {
		switch {
		case activation.Skipped:
			skipped++
			output.printf("○ %-20s skipped — %s\n", activation.ID, escapeTerminal(activation.Detail))
		case activation.State != "":
			output.printf("× %-20s %-15s\n", activation.ID, activation.State)
			output.printf("  Expected  %s\n  Live      %s\n", escapeTerminal(activation.Expected), escapeTerminal(activation.Live))
			if activation.Detail != "" {
				output.printf("  %s\n", escapeTerminal(activation.Detail))
			}
		}
	}
	checkCount := len(report.Pairs) + len(report.Activations)
	if report.Clean() {
		output.printf("✓ clean — %d checks, %d files", checkCount, files)
		if skipped > 0 {
			output.printf(", %d skipped", skipped)
		}
		output.println()
		return output.err
	}
	output.printf("\n%d drift item(s) across %d check(s)\n", report.FindingCount(), checkCount)
	return output.err
}

func (a *application) printSyncResult(plan link.Plan, verification link.Report) error {
	return writeJSON(a.streams.Out, struct {
		Plan         link.Plan    `json:"plan"`
		Applied      bool         `json:"applied"`
		Verification reportOutput `json:"verification"`
	}{plan, true, newReportOutput(verification)})
}

func (a *application) printPlan(plan link.Plan, applying bool) error {
	if a.global.format == "json" {
		return writeJSON(a.streams.Out, plan)
	}
	output := printer{writer: a.streams.Out}
	for _, operation := range plan.Operations {
		switch operation.Kind {
		case link.OperationCopy:
			output.printf("COPY    %s:%s\n        %s\n     →  %s\n", operation.Pair, escapeTerminal(displayRelative(operation.Relative)), escapeTerminal(operation.Source), escapeTerminal(operation.Target))
			if operation.Detail != "" {
				output.printf("        ! %s\n", escapeTerminal(operation.Detail))
			}
		case link.OperationDelete:
			output.printf("DELETE  %s:%s\n        %s\n", operation.Pair, escapeTerminal(displayRelative(operation.Relative)), escapeTerminal(operation.Target))
		case link.OperationMkdir:
			output.printf("MKDIR   %s\n", escapeTerminal(operation.Target))
		}
	}
	for _, finding := range plan.Unresolved {
		output.printf("BLOCKED %s:%s — %s\n", finding.Pair, escapeTerminal(displayRelative(finding.Relative)), escapeTerminal(finding.Detail))
	}
	switch {
	case len(plan.Operations) == 0 && len(plan.Unresolved) == 0:
		output.println("✓ already clean — nothing to sync")
	case !applying:
		output.printf("\nPlan only: %d operation(s). Add --apply to write.\n", len(plan.Operations))
	default:
		output.printf("\nApplying %d operation(s)…\n", len(plan.Operations))
	}
	return output.err
}

func (a *application) printGuard(violations []link.Violation, agent string, reminder bool) error {
	if a.global.format == "json" && agent == "human" {
		if violations == nil {
			violations = make([]link.Violation, 0)
		}
		return writeJSON(a.streams.Out, struct {
			Blocked    bool             `json:"blocked"`
			Violations []link.Violation `json:"violations"`
		}{!reminder && len(violations) > 0, violations})
	}
	var text strings.Builder
	contextOutput := printer{writer: &text}
	if len(violations) > 0 {
		if reminder {
			contextOutput.printf("agentlink sync reminder:\n")
		} else {
			contextOutput.printf("agentlink blocked this change:\n")
		}
		for _, violation := range violations {
			contextOutput.printf("- %s:%s: %s", violation.Pair, escapeTerminal(displayRelative(violation.Relative)), escapeTerminal(violation.Message))
			if violation.Counterpart != "" {
				contextOutput.printf("; update %s", escapeTerminal(violation.Counterpart))
			}
			contextOutput.printf("\n")
		}
	}
	if contextOutput.err != nil {
		return contextOutput.err
	}
	if agent != "human" && reminder {
		return writeJSON(a.streams.Out, map[string]any{
			"hookSpecificOutput": map[string]string{
				"hookEventName": "PostToolUse", "additionalContext": strings.TrimSpace(text.String()),
			},
		})
	}
	if text.Len() == 0 {
		return nil
	}
	output := a.streams.Err
	if reminder || agent != "human" {
		output = a.streams.Out
	}
	_, err := fmt.Fprint(output, text.String())
	return err
}

func (a *application) flagError(command string, err error) error {
	if errors.Is(err, flag.ErrHelp) {
		output := printer{writer: a.streams.Out}
		output.printf("Run `agentlink help` for %s usage.\n", command)
		return output.err
	}
	return a.usageError(err.Error())
}

func (a *application) usageError(message string) error {
	return &ExitError{Code: exitUsage, Err: errors.New(message)}
}

func (a *application) printHelp() error {
	_, err := io.WriteString(a.streams.Out, `agentlink — keep coding-agent peer artifacts aligned

Usage:
  agentlink [global flags] <command> [flags]

Commands:
  check       Detect missing or drifting peer artifacts (default; aliases: status, audit)
  sync        Preview or apply an explicit one-way reconciliation
  adopt       Move project configuration into .agents and link it back
  guard       Block when provided changed paths currently drift
  remind      Emit agent-hook context for provided changed paths
  list        Show resolved sources, pairs, and intentional divergences
  validate    Strictly validate YAML configuration
  doctor      Validate configuration and safely open every source
  schema      Print the bundled JSON Schema
  init        Create a minimal config and local editor schema
  version     Print the version

Global flags:
  -c, --config PATH    Configuration path (also AGENTLINK_CONFIG)
      --format FORMAT  human or json
      --json           Shortcut for --format json
  -q, --quiet          Suppress normal output

Safe sync:
  agentlink sync --from agents            # preview only
  agentlink sync --from agents --apply    # copy missing/drifting files
  agentlink sync --from claude --prune --apply

Adopt existing configuration:
  agentlink adopt --from .claude/skills
  agentlink adopt --from .claude/skills --apply
  agentlink adopt --from .codex/skills --apply --force

Provider-neutral guard:
  printf '%s\n' AGENTS.md | agentlink guard
  agentlink guard AGENTS.md .claude/skills/review/SKILL.md

Exit codes: 0 clean/success, 1 drift or blocked guard, 2 usage error.
`)
	return err
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	return set
}

func selectedPairs(cfg *config.Config, values []string, includeMCP bool) (map[string]bool, error) {
	valid := cfg.ArtifactPairIDs()
	if includeMCP {
		valid = cfg.PairIDs()
	}
	selected := make(map[string]bool, len(values))
	for _, value := range values {
		if !slices.Contains(valid, value) {
			return nil, fmt.Errorf("unknown pair %q; choose one of: %s", value, strings.Join(valid, ", "))
		}
		selected[value] = true
	}
	return selected, nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

type printer struct {
	writer io.Writer
	err    error
}

func (p *printer) printf(format string, values ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.writer, format, values...)
}

func (p *printer) println(values ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintln(p.writer, values...)
}

func displayRelative(relative string) string {
	if relative == "." {
		return "(root)"
	}
	return relative
}

// escapeTerminal renders repository- and configuration-controlled text so
// terminal control bytes are visibly escaped rather than interpreted.
// Filesystem names, configured paths, and diagnostic text cross an
// attacker-controlled input boundary, so they must be data-only at human and
// agent-hook output sinks (CWE-150).
func escapeTerminal(value string) string {
	if !hasTerminalControl(value) {
		return value
	}
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte (for example a bare C1 byte such as OSC).
			// Filenames are byte strings, not necessarily valid UTF-8.
			if c := value[i]; isTerminalControlByte(c) {
				fmt.Fprintf(&b, `\x%02x`, c)
			} else {
				b.WriteByte(c)
			}
			i++
			continue
		}
		if isTerminalControl(r) {
			fmt.Fprintf(&b, `\x%02x`, r)
		} else {
			b.WriteString(value[i : i+size])
		}
		i += size
	}
	return b.String()
}

func escapePaths(values []string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = escapeTerminal(value)
	}
	return out
}

func hasTerminalControl(value string) bool {
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			if isTerminalControlByte(value[i]) {
				return true
			}
			i++
			continue
		}
		if isTerminalControl(r) {
			return true
		}
		i += size
	}
	return false
}

func isTerminalControl(r rune) bool {
	// C0 controls (0x00-0x1F) include ESC, which begins terminal escape
	// sequences; DEL is 0x7F. C1 controls (0x80-0x9F code points) carry
	// OSC/CSI/APC terminators used to rewrite or query terminal state.
	return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f)
}

func isTerminalControlByte(c byte) bool {
	return c < 0x20 || c == 0x7f || (c >= 0x80 && c <= 0x9f)
}

func mapsKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
