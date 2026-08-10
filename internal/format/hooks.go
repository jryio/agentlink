package format

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
)

const canonicalCommandAgent = "agent"

// Canonicalize converts a native hooks document into the canonical .agents
// hooks.json form: a bare event → matcher-groups map with canonical event
// names, timeouts in seconds, and agent-neutral command tokens. Events with
// no canonical mapping stay verbatim so genuinely target-only hooks surface
// as drift. Malformed event entries are an error — silently dropping them
// would hide exactly the breakage comparison exists to catch.
func (hookFormatter) Canonicalize(self agent.Spec, data []byte) ([]byte, error) {
	doc, err := DecodeDocument(self.Hooks.Format, data)
	if err != nil {
		return nil, err
	}
	events, err := extractEvents(self, doc)
	if err != nil {
		return nil, err
	}
	canonical, err := canonicalizeEvents(self, events)
	if err != nil {
		return nil, err
	}
	return encodeJSON(canonical)
}

// CanonicalHookDocument canonicalizes one peer's hook file for comparison:
// events the other peer cannot express are dropped from the canonical form
// (the translate path warns about them at sync time). Comparison normalizes
// both peers through this function.
func CanonicalHookDocument(self, other agent.Spec, data []byte) ([]byte, error) {
	canonical, err := (hookFormatter{}).Canonicalize(self, data)
	if err != nil {
		return nil, err
	}
	events, err := DecodeDocument(agent.DialectJSON, canonical)
	if err != nil {
		return nil, err
	}
	for event := range events {
		if !slices.Contains(agent.CanonicalEvents, event) {
			continue // verbatim target-only event: never filtered
		}
		if _, supported := other.HookEvent(event); !supported {
			delete(events, event)
		}
	}
	return encodeJSON(events)
}

// hookFormatter translates the canonical hooks document into a target's
// native shape, merging into settings documents instead of replacing them.
type hookFormatter struct{}

func (hookFormatter) Kind() string { return "hook" }

func (hookFormatter) Format(canonical, existing []byte, target agent.Spec) ([]byte, []string, error) {
	events, err := DecodeDocument(agent.DialectJSON, canonical)
	if err != nil {
		return nil, nil, fmt.Errorf("parse canonical hooks: %w", err)
	}
	var warnings []string
	out := make(map[string]any, len(events))
	for _, event := range sortedKeys(events) {
		name, supported := target.HookEvent(event)
		if !supported {
			warnings = append(warnings, "drop unsupported hook event "+event+" for "+target.ID)
			continue
		}
		converted, err := emitEvent(target, events[event])
		if err != nil {
			return nil, nil, fmt.Errorf("hook event %s: %w", event, err)
		}
		if converted != nil {
			out[name] = converted
		}
	}

	switch target.Hooks.Wrapper {
	case agent.WrapperBare:
		encoded, err := EncodeDocument(target.Hooks.Format, out)
		return encoded, warnings, err
	case agent.WrapperHooks:
		wrapped := map[string]any{"hooks": out}
		if target.Hooks.Version > 0 {
			wrapped["version"] = target.Hooks.Version
		}
		encoded, err := EncodeDocument(target.Hooks.Format, wrapped)
		return encoded, warnings, err
	case agent.WrapperSettings:
		settings := map[string]any{}
		if len(existing) > 0 {
			parsed, err := DecodeDocument(target.Hooks.Format, existing)
			if err != nil {
				return nil, nil, fmt.Errorf("merge into %s settings: existing file is invalid: %w", target.ID, err)
			}
			settings = parsed
		}
		if target.Hooks.Shape == agent.ShapeList {
			settings["hooks"] = wrapList(out)
		} else {
			settings["hooks"] = out
		}
		encoded, err := EncodeDocument(target.Hooks.Format, settings)
		return encoded, warnings, err
	}
	return nil, nil, fmt.Errorf("unsupported hooks wrapper %q", target.Hooks.Wrapper)
}

// emitEvent converts one canonical event's matcher groups into the target's
// per-event shape. A nil result means the event has no handlers and is
// dropped rather than emitted empty.
func emitEvent(target agent.Spec, value any) (any, error) {
	groups, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("groups must be an array")
	}
	if target.Hooks.Shape == agent.ShapeGroups {
		out := make([]any, 0, len(groups))
		for _, group := range groups {
			matcher, handlers, err := emitGroup(target, group)
			if err != nil {
				return nil, err
			}
			emitted := map[string]any{"hooks": handlers}
			if matcher != "" {
				emitted["matcher"] = matcher
			}
			out = append(out, emitted)
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	}
	entries := make([]any, 0, len(groups))
	for _, group := range groups {
		matcher, handlers, err := emitGroup(target, group)
		if err != nil {
			return nil, err
		}
		for _, handler := range handlers {
			entry, ok := handler.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("handler must be an object")
			}
			switch {
			case matcher == "":
			case target.Hooks.MatcherShape == agent.MatcherToolName:
				entry["matcher"] = map[string]any{"tool_name": matcher}
			default:
				entry["matcher"] = matcher
			}
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return entries, nil
}

// emitGroup extracts a canonical group's matcher ("" when absent or "*") and
// converts its handlers for the target.
func emitGroup(target agent.Spec, group any) (string, []any, error) {
	groupMap, ok := group.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("group must be an object")
	}
	matcher, _ := groupMap["matcher"].(string)
	if matcher == "*" {
		matcher = ""
	}
	nested, _ := groupMap["hooks"].([]any)
	handlers := make([]any, 0, len(nested))
	for _, handler := range nested {
		handlerMap, ok := handler.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("handler must be an object")
		}
		handlers = append(handlers, emitHandler(target, handlerMap))
	}
	return matcher, handlers, nil
}

// emitHandler copies one canonical handler, rewriting agent-neutral command
// tokens to the target's ID and rescaling the timeout field. The canonical
// hub keeps the neutral tokens: its document is the store every target
// translates from, never an executable hook file.
func emitHandler(target agent.Spec, handler map[string]any) map[string]any {
	out := make(map[string]any, len(handler))
	for key, value := range handler {
		switch key {
		case "timeout":
			out[targetTimeoutField(target)] = scaleTimeout(value, target, true)
		case "command":
			if command, ok := value.(string); ok && target.ID != agent.HubID {
				value = rewriteCommandAgent(command, canonicalCommandAgent, target.ID)
			}
			out[key] = value
		default:
			out[key] = value
		}
	}
	return out
}

// canonicalizeEvents renames events to canonical form and folds every source
// shape into matcher groups. An event whose entries fail shape validation is
// an error, not a silent skip.
func canonicalizeEvents(spec agent.Spec, events map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(events))
	for _, event := range sortedKeys(events) {
		canonical, ok := spec.HookCanonical(event)
		if !ok {
			canonical = event // target-only event: keep verbatim so drift stays visible
		}
		groups, err := canonicalGroups(spec, events[event])
		if err != nil {
			return nil, fmt.Errorf("hook event %s: %w", event, err)
		}
		if len(groups) == 0 {
			continue
		}
		out[canonical] = groups
	}
	return out, nil
}

func canonicalGroups(spec agent.Spec, value any) ([]any, error) {
	entries, ok := asSlice(value)
	if !ok {
		return nil, fmt.Errorf("hook event entries must be an array")
	}
	if spec.Hooks.Shape == agent.ShapeGroups {
		groups := make([]any, 0, len(entries))
		for _, entry := range entries {
			group, ok := entry.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("hook group must be an object")
			}
			out := map[string]any{}
			if matcher, _ := group["matcher"].(string); matcher != "" && matcher != "*" {
				out["matcher"] = matcher
			}
			nested, ok := asSlice(group["hooks"])
			if !ok {
				return nil, fmt.Errorf("hook group hooks must be an array")
			}
			handlers := make([]any, 0, len(nested))
			for _, handler := range nested {
				handlerMap, ok := handler.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("hook handler must be an object")
				}
				handlers = append(handlers, canonicalHandler(spec, handlerMap))
			}
			out["hooks"] = handlers
			groups = append(groups, out)
		}
		return groups, nil
	}
	byMatcher := make(map[string][]any)
	var order []string
	for _, entry := range entries {
		handler, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("hook entry must be an object")
		}
		matcher := ""
		switch raw := handler["matcher"].(type) {
		case string:
			matcher = raw
		case map[string]any:
			matcher, _ = raw["tool_name"].(string)
		}
		if matcher == "*" {
			matcher = ""
		}
		if _, seen := byMatcher[matcher]; !seen {
			order = append(order, matcher)
		}
		byMatcher[matcher] = append(byMatcher[matcher], canonicalHandler(spec, handler))
	}
	slices.Sort(order)
	groups := make([]any, 0, len(order))
	for _, matcher := range order {
		group := map[string]any{"hooks": byMatcher[matcher]}
		if matcher != "" {
			group["matcher"] = matcher
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// canonicalHandler copies one source handler into canonical form: timeout in
// seconds under the canonical key, agent-specific command tokens neutralized,
// structural keys (matcher, event) re-derived from the group shape.
func canonicalHandler(spec agent.Spec, handler map[string]any) map[string]any {
	out := make(map[string]any, len(handler))
	timeoutField := targetTimeoutField(spec)
	for key, value := range handler {
		switch key {
		case "matcher", "event":
			// structural keys are re-derived from the group shape
			continue
		case timeoutField:
			out["timeout"] = scaleTimeout(value, spec, false)
		case "command":
			if command, ok := value.(string); ok {
				value = rewriteCommandAgent(command, spec.ID, canonicalCommandAgent)
			}
			out[key] = value
		default:
			out[key] = value
		}
	}
	return out
}

func rewriteCommandAgent(command, from, to string) string {
	command = replaceCommandToken(command, "--agent "+from, "--agent "+to)
	return replaceCommandToken(command, "remind-"+from, "remind-"+to)
}

func replaceCommandToken(command, from, to string) string {
	var out strings.Builder
	last := 0
	search := 0
	replaced := false
	for search < len(command) {
		offset := strings.Index(command[search:], from)
		if offset < 0 {
			break
		}
		start := search + offset
		end := start + len(from)
		if (start == 0 || !isAgentIDByte(command[start-1])) &&
			(end == len(command) || !isAgentIDByte(command[end])) {
			if !replaced {
				out.Grow(len(command))
				replaced = true
			}
			out.WriteString(command[last:start])
			out.WriteString(to)
			last = end
			search = end
			continue
		}
		search = start + 1
	}
	if !replaced {
		return command
	}
	out.WriteString(command[last:])
	return out.String()
}

func isAgentIDByte(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		value == '.' || value == '_' || value == '-'
}

func targetTimeoutField(spec agent.Spec) string {
	if spec.Hooks.TimeoutField != "" {
		return spec.Hooks.TimeoutField
	}
	return "timeout"
}

// scaleTimeout converts timeout values between canonical seconds and target
// units. toTarget multiplies by the target's unit scale; otherwise it
// divides.
func scaleTimeout(value any, spec agent.Spec, toTarget bool) any {
	scale := 1
	if spec.Hooks.TimeoutUnit == agent.TimeoutMilliseconds {
		scale = 1000
	}
	if scale == 1 {
		// DecodeDocument yields json.Number, which go.yaml.in/yaml/v3
		// marshals as a quoted string — normalize to int64/float64 so YAML
		// targets (hermes) receive numeric timeouts.
		if number, ok := value.(json.Number); ok {
			if i, err := number.Int64(); err == nil {
				return i
			}
			if f, err := number.Float64(); err == nil {
				return f
			}
		}
		return value
	}
	number, err := numberFloat(value)
	if err != nil {
		return value
	}
	if toTarget {
		number *= float64(scale)
	} else {
		number /= float64(scale)
	}
	if number == float64(int64(number)) {
		return json.Number(strconv.FormatInt(int64(number), 10))
	}
	return json.Number(strconv.FormatFloat(number, 'g', -1, 64))
}

func numberFloat(value any) (float64, error) {
	switch number := value.(type) {
	case json.Number:
		return number.Float64()
	case float64:
		return number, nil
	case int:
		return float64(number), nil
	case int64:
		return float64(number), nil
	}
	return 0, fmt.Errorf("not a number: %v", value)
}

// extractEvents unwraps the hook event map from a parsed document. The
// ShapeList (TOML [[hooks]]) form is regrouped into an event → entries map.
func extractEvents(spec agent.Spec, doc map[string]any) (map[string]any, error) {
	var raw any = doc
	if spec.Hooks.Wrapper != agent.WrapperBare {
		raw = doc["hooks"]
		if raw == nil {
			return map[string]any{}, nil
		}
	}
	if spec.Hooks.Shape == agent.ShapeList {
		entries, ok := asSlice(raw)
		if !ok {
			return nil, fmt.Errorf("hooks must be an array of tables")
		}
		grouped := make(map[string]any)
		for _, entry := range entries {
			handler, ok := entry.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("hook entry must be an object")
			}
			event, _ := handler["event"].(string)
			if event == "" {
				return nil, fmt.Errorf("hook entry is missing its event name")
			}
			grouped[event] = append(groupedEvent(grouped[event]), handler)
		}
		return grouped, nil
	}
	events, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hooks must be an object keyed by event")
	}
	return events, nil
}

func groupedEvent(value any) []any {
	entries, _ := value.([]any)
	return entries
}

// asSlice coerces decoded arrays: TOML array-of-tables decode as
// []map[string]any rather than []any.
func asSlice(value any) ([]any, bool) {
	switch entries := value.(type) {
	case []any:
		return entries, true
	case []map[string]any:
		out := make([]any, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entry)
		}
		return out, true
	}
	return nil, false
}

// wrapList converts an event → entries map into kimi's [[hooks]] array form.
func wrapList(events map[string]any) []any {
	list := make([]any, 0, len(events))
	for _, event := range sortedKeys(events) {
		entries, _ := events[event].([]any)
		for _, entry := range entries {
			handler, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			withEvent := make(map[string]any, len(handler)+1)
			withEvent["event"] = event
			for key, value := range handler {
				withEvent[key] = value
			}
			list = append(list, withEvent)
		}
	}
	return list
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func init() {
	Register(hookFormatter{})
}
