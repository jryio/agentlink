package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"

	"github.com/jryio/agentlink/internal/agent"
)

// CanonicalHookDocument parses a target agent's hook file and emits the
// canonical .agents hooks.json form: a bare event → matcher-groups map with
// canonical event names, timeouts in seconds, and agent-neutral command
// tokens. Comparison normalizes both peers through this function. Events the
// other peer cannot express are dropped from the canonical form (the
// translate path warns about them at sync time); events with no canonical
// mapping stay verbatim so genuinely target-only hooks surface as drift.
func CanonicalHookDocument(self, other agent.Spec, data []byte) ([]byte, error) {
	doc, err := decodeDocument(self.HooksFormat, data)
	if err != nil {
		return nil, err
	}
	events, err := extractEvents(self, doc)
	if err != nil {
		return nil, err
	}
	canonical := canonicalizeEvents(self, events)
	for event := range canonical {
		if !slices.Contains(agent.CanonicalEvents, event) {
			continue // verbatim target-only event: never filtered
		}
		if _, supported := other.HookEvent(event); !supported {
			delete(canonical, event)
		}
	}
	return encodeJSON(canonical)
}

// hookFormatter translates the canonical hooks document into a target's
// native shape, merging into settings documents instead of replacing them.
type hookFormatter struct{}

func (hookFormatter) Kind() string { return "hook" }

func (hookFormatter) Format(canonical, existing []byte, target agent.Spec) ([]byte, []string, error) {
	var events map[string]any
	if err := json.Unmarshal(canonical, &events); err != nil {
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

	switch target.HooksWrapper {
	case agent.WrapperBare:
		return encodeDocument(target.HooksFormat, out, warnings)
	case agent.WrapperHooks:
		wrapped := map[string]any{"hooks": out}
		if target.HooksVersion > 0 {
			wrapped["version"] = target.HooksVersion
		}
		return encodeDocument(target.HooksFormat, wrapped, warnings)
	case agent.WrapperSettings:
		settings := map[string]any{}
		if len(existing) > 0 {
			parsed, err := decodeDocument(target.HooksFormat, existing)
			if err != nil {
				return nil, nil, fmt.Errorf("merge into %s settings: existing file is invalid: %w", target.ID, err)
			}
			settings = parsed
		}
		if target.HooksShape == agent.ShapeList {
			settings["hooks"] = wrapList(out)
		} else {
			settings["hooks"] = out
		}
		return encodeDocument(target.HooksFormat, settings, warnings)
	}
	return nil, nil, fmt.Errorf("unsupported hooks wrapper %q", target.HooksWrapper)
}

// emitEvent converts one canonical event's matcher groups into the target's
// per-event shape. A nil result means the event has no handlers and is
// dropped rather than emitted empty.
func emitEvent(target agent.Spec, value any) (any, error) {
	groups, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("groups must be an array")
	}
	if target.HooksShape == agent.ShapeGroups {
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
			case target.HooksMatcherShape == "tool_name":
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
// tokens to the target's ID and rescaling the timeout field.
func emitHandler(target agent.Spec, handler map[string]any) map[string]any {
	out := make(map[string]any, len(handler))
	for key, value := range handler {
		switch key {
		case "timeout":
			out[targetTimeoutField(target)] = scaleTimeout(value, target, true)
		case "command":
			if command, ok := value.(string); ok {
				value = strings.NewReplacer(
					"--agent agent", "--agent "+target.ID,
					"remind-agent", "remind-"+target.ID,
				).Replace(command)
			}
			out[key] = value
		default:
			out[key] = value
		}
	}
	return out
}

// canonicalizeEvents renames events to canonical form and folds every source
// shape into matcher groups.
func canonicalizeEvents(spec agent.Spec, events map[string]any) map[string]any {
	out := make(map[string]any, len(events))
	for _, event := range sortedKeys(events) {
		canonical, ok := spec.HookCanonical(event)
		if !ok {
			canonical = event // target-only event: keep verbatim so drift stays visible
		}
		groups, err := canonicalGroups(spec, events[event])
		if err != nil || len(groups) == 0 {
			continue
		}
		out[canonical] = groups
	}
	return out
}

func canonicalGroups(spec agent.Spec, value any) ([]any, error) {
	entries, ok := asSlice(value)
	if !ok {
		return nil, fmt.Errorf("hook event entries must be an array")
	}
	if spec.HooksShape == agent.ShapeGroups {
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
			nested, _ := group["hooks"].([]any)
			handlers := make([]any, 0, len(nested))
			for _, handler := range nested {
				if handlerMap, ok := handler.(map[string]any); ok {
					handlers = append(handlers, canonicalHandler(spec, handlerMap))
				}
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
				value = strings.NewReplacer(
					"--agent "+spec.ID, "--agent agent",
					"remind-"+spec.ID, "remind-agent",
				).Replace(command)
			}
			out[key] = value
		default:
			out[key] = value
		}
	}
	return out
}

func targetTimeoutField(spec agent.Spec) string {
	if spec.HooksTimeoutField != "" {
		return spec.HooksTimeoutField
	}
	return "timeout"
}

// scaleTimeout converts timeout values between canonical seconds and target
// units. toTarget multiplies by the spec scale; otherwise it divides.
func scaleTimeout(value any, spec agent.Spec, toTarget bool) any {
	scale := spec.HooksTimeoutScale
	if scale <= 1 {
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
	if spec.HooksWrapper != agent.WrapperBare {
		raw = doc["hooks"]
		if raw == nil {
			return map[string]any{}, nil
		}
	}
	if spec.HooksShape == agent.ShapeList {
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

func decodeDocument(format agent.HookFormat, data []byte) (map[string]any, error) {
	switch format {
	case agent.HookFormatJSON:
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		result := make(map[string]any)
		if err := decoder.Decode(&result); err != nil {
			return nil, fmt.Errorf("decode hook JSON: %w", err)
		}
		return result, nil
	case agent.HookFormatTOML:
		result := make(map[string]any)
		if _, err := toml.Decode(string(data), &result); err != nil {
			return nil, fmt.Errorf("decode hook TOML: %w", err)
		}
		return result, nil
	case agent.HookFormatYAML:
		result := make(map[string]any)
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("decode hook YAML: %w", err)
		}
		return result, nil
	}
	return nil, fmt.Errorf("unsupported hook format %q", format)
}

func encodeDocument(format agent.HookFormat, doc map[string]any, warnings []string) ([]byte, []string, error) {
	switch format {
	case agent.HookFormatJSON:
		out, err := encodeJSON(doc)
		return out, warnings, err
	case agent.HookFormatTOML:
		var buffer bytes.Buffer
		if err := toml.NewEncoder(&buffer).Encode(doc); err != nil {
			return nil, nil, fmt.Errorf("encode hook TOML: %w", err)
		}
		return buffer.Bytes(), warnings, nil
	case agent.HookFormatYAML:
		out, err := yaml.Marshal(doc)
		if err != nil {
			return nil, nil, fmt.Errorf("encode hook YAML: %w", err)
		}
		return out, warnings, nil
	}
	return nil, nil, fmt.Errorf("unsupported hook format %q", format)
}

// encodeJSON emits a deterministic, sorted, indented JSON document.
func encodeJSON(value any) ([]byte, error) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode hook JSON: %w", err)
	}
	return append(out, '\n'), nil
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
