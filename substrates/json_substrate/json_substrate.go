package json_substrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ahmetson/mushroom"
)

type Substrate struct {
	url mushroom.Hypha
	mu  sync.RWMutex
}

var _ mushroom.Substrate = (*Substrate)(nil)

type Mycelium struct {
	url       mushroom.Hypha
	data      any
	soil      *mushroom.Soil
	substrate mushroom.Substrate
}

var _ mushroom.Mycelium = (*Mycelium)(nil)

// New returns a JSON substrate registered with the pattern pkg:json/$#$.json.
func New() mushroom.Substrate {
	substrate := &Substrate{}
	soil := &mushroom.Soil{}
	substrate.url, _ = soil.Hypha("pkg:json/$#$.json")
	return substrate
}

// Root creates the initial mycelium colony by foraging the given path and then
// digesting the content into a mycelium network with the initial substrate and soil.
// Optional substrates are registered in the soil before germination.
//
// Example:
//
//	mycelium, err := json_substrate.Root("pkg:json/configs#app.json")
//	mycelium, err := json_substrate.Root("pkg:json/configs#app.json", otherSubstrate)
func Root(mushroomURL string, substrates ...mushroom.Substrate) (*Mycelium, error) {
	substrate := New()
	soil := &mushroom.Soil{}
	for _, s := range substrates {
		if err := soil.AddSubstrate(s); err != nil {
			return nil, err
		}
	}
	hypha, err := soil.Hypha(mushroomURL)
	if err != nil {
		return nil, err
	}
	got, err := soil.Germinate(hypha, substrate)
	if err != nil {
		return nil, err
	}
	return got.(*Mycelium), nil
}

func (substrate *Substrate) Digest(url mushroom.Hypha, data any, soil *mushroom.Soil) (mushroom.Mycelium, error) {
	if !url.URL {
		return nil, fmt.Errorf("json substrate: digest URL must be a Mushroom URL")
	}
	if url.Dereference {
		return nil, fmt.Errorf("json substrate: digest URL must be a link")
	}
	if !substrate.url.Satisfies(url) {
		return nil, fmt.Errorf("json substrate: digest URL %q does not satisfy %q", url.String(), substrate.url.String())
	}

	decoded, err := decode(data)
	if err != nil {
		return nil, err
	}

	substrateInterface := mushroom.Substrate(substrate)
	mycelium := &Mycelium{
		url:       url,
		data:      decoded,
		soil:      soil,
		substrate: substrateInterface,
	}
	return mycelium, nil
}

func (substrate *Substrate) MushroomURL() string {
	return substrate.url.String()
}

// Forage reads the JSON file identified by mushroomURL and returns its raw
// content as a string. The file path is formed by joining PackageID and
// ModuleID with filepath.Join. The returned string can be passed directly to
// Digest. Reading is protected by a read lock so concurrent Forage calls do
// not race with a concurrent Sow.
//
//	raw, err := substrate.Forage(soil.Hypha("pkg:json/configs#app.json"))
//	mycelium, err := substrate.Digest("pkg:json/configs#app.json", raw, soil)
func (substrate *Substrate) Forage(url mushroom.Hypha) (any, error) {
	if !substrate.url.Satisfies(url) {
		return nil, fmt.Errorf("json substrate: forage URL %q does not satisfy pattern %q", url.String(), substrate.url.String())
	}
	path := filepath.Join(url.PackageID, url.ModuleID)

	substrate.mu.RLock()
	data, err := os.ReadFile(path)
	substrate.mu.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("json substrate: forage %q: %w", path, err)
	}
	return string(data), nil
}

// Sow writes nutrients to the JSON file identified by mushroomURL.
// The file path is formed by joining PackageID and ModuleID with filepath.Join.
// data may be a string (written verbatim) or any JSON-marshalable value
// (marshaled with two-space indentation). Writing is protected by a write lock
// so concurrent Sow calls and Forage calls do not race.
//
//	err := substrate.Sow(soil.Hypha("pkg:json/configs#app.json"), updatedData)
func (substrate *Substrate) Sow(url mushroom.Hypha, data any) error {
	if !substrate.url.Satisfies(url) {
		return fmt.Errorf("json substrate: sow URL %q does not satisfy pattern %q", url.String(), substrate.url.String())
	}
	path := filepath.Join(url.PackageID, url.ModuleID)

	var raw []byte
	switch v := data.(type) {
	case string:
		raw = []byte(v)
	default:
		var err error
		raw, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("json substrate: sow marshal: %w", err)
		}
	}

	substrate.mu.Lock()
	err := os.WriteFile(path, raw, 0o644)
	substrate.mu.Unlock()

	if err != nil {
		return fmt.Errorf("json substrate: sow %q: %w", path, err)
	}
	return nil
}

// Link normalizes path into an absolute Mushroom link.
// Non-Mushroom symbols are returned unchanged.
// Dereference paths (*) are rejected.
//
// Missing or wildcard ($) fields — type, package, module — are filled from the
// mycelium's own URL so partial paths can be resolved:
//
//	// mycelium URL: pkg:json$#config.json
//	link, err := mycelium.Link("pkg:$?var=port")
//	// "pkg:json$#config.json?var=port"
//
//	link, err := mycelium.Link("pkg:json/github.com/example/config#config.json?var=port")
//	// "pkg:json/github.com/example/config#config.json?var=port"
//
//	link, err := mycelium.Link("port")
//	// "port"  (symbol, returned as-is)
//
// Fails when the path contains a dereference, is not recognized by the soil,
// or refers to a resource that does not exist in this mycelium's data:
//
//	_, err := mycelium.Link("pkg:$?*var=port")
//	_, err := mycelium.Link("*pkg:json$#config.json?var=port")
//	_, err := mycelium.Link("pkg:$?var=nonExistent")
func (mycelium *Mycelium) Link(path string) (string, error) {
	hypha, err := mycelium.soil.Hypha(path, mycelium.url)
	if err != nil {
		return "", err
	}
	if hypha.Dereference {
		return "", errors.New("json substrate: link cannot contain a dereference")
	}

	// Non-Mushroom symbols are passed through as-is.
	if !hypha.URL {
		return path, nil
	}

	// Validate that the resolved link is recognized by the soil
	// (either a loaded colony or a registered substrate).
	if _, _, err := mycelium.soil.Recognize(hypha); err != nil {
		return "", err
	}

	// Recognize only validates the URL pattern. When the link refers to this
	// mycelium's own module and has a resource path, also verify that the
	// resource actually exists in the data.
	if hypha.ResourceKind != "" && mycelium.url.Satisfies(hypha) {
		if _, err := lookup(mycelium.data, hypha.ResourcePath); err != nil {
			return "", fmt.Errorf("json substrate: resource %q not found: %w", hypha.ResourcePath.String(), err)
		}
	}

	return hypha.String(), nil
}

func (mycelium *Mycelium) Spore(path string) (any, error) {
	// Resolve the path, loading any unrecognized mycelia referenced by
	// dereference lambdas. Each iteration loads one mycelium and retries, so
	// paths with multiple unloaded lambdas are handled progressively.
	var hypha mushroom.Hypha
	for {
		var err error
		hypha, err = mycelium.soil.Hypha(path, mycelium.url)
		if err == nil {
			break
		}
		var unrecognized *mushroom.ErrUnrecognizedMycelium
		if !errors.As(err, &unrecognized) {
			return nil, err
		}
		if _, germinateErr := mycelium.soil.Germinate(unrecognized.Hypha, unrecognized.Substrate); germinateErr != nil {
			return nil, fmt.Errorf("json substrate: spore %q: %w", path, germinateErr)
		}
		// Colony is now registered; retry soil.Hypha.
	}

	if !hypha.URL {
		return path, nil
	}
	if !hypha.Dereference {
		return nil, fmt.Errorf("json substrate: spore requires a dereference URL, got link %q", path)
	}
	if hypha.DereferenceType != mushroom.DereferenceTypeResource {
		return nil, fmt.Errorf("json substrate: spore requires a resource dereference, got %q", hypha.DereferenceType)
	}
	if hypha.ResourceKind != mushroom.ResourceKindVar {
		return nil, fmt.Errorf("json substrate: unsupported resource kind %q", hypha.ResourceKind)
	}

	colony, substrate, recognizeErr := mycelium.soil.Recognize(hypha)
	if recognizeErr != nil {
		return nil, fmt.Errorf("json substrate: spore %q: %w", path, recognizeErr)
	}

	// This mycelium owns the URL — look up directly in our own data.
	if colony == mycelium {
		return lookup(mycelium.data, hypha.ResourcePath)
	}

	// A different colony is loaded — delegate to it.
	if colony != nil {
		return colony.Spore(path)
	}

	// No loaded colony — germinate on demand, then delegate.
	m, germinateErr := mycelium.soil.Germinate(hypha, substrate)
	if germinateErr != nil {
		return nil, fmt.Errorf("json substrate: spore %q: %w", path, germinateErr)
	}
	return m.Spore(path)
}

func (mycelium *Mycelium) Fruit(value any) (any, error) {
	switch typed := value.(type) {
	case string:
		hypha, err := mycelium.soil.Hypha(typed)
		if err != nil {
			return typed, nil
		}
		if hypha.URL && hypha.Dereference {
			return mycelium.Spore(typed)
		}

		return typed, nil
	case map[string]any:
		clone := make(map[string]any, len(typed))
		for key, item := range typed {
			fruited, err := mycelium.Fruit(item)
			if err != nil {
				return nil, err
			}
			clone[key] = fruited
		}

		return clone, nil
	case []any:
		clone := make([]any, len(typed))
		for index, item := range typed {
			fruited, err := mycelium.Fruit(item)
			if err != nil {
				return nil, err
			}
			clone[index] = fruited
		}

		return clone, nil
	default:
		return value, nil
	}
}

func (mycelium *Mycelium) Mineralize() (any, error) {
	mineralized, err := json.Marshal(mycelium.data)
	if err != nil {
		return nil, err
	}

	return string(mineralized), nil
}

// Inoculate overwrites the value at path with value.
//
// The path uses the same pkg:$?var=… syntax as Spore. Wildcards in the
// module URL are resolved from this mycelium; when the path names a different
// module the mutation is delegated to that colony (germinating it on demand).
// It supports:
//   - Simple field: pkg:$?var=services[0].port — sets the port field on the first service.
//   - Indexed element: pkg:$?var=services[0] — replaces the first service entirely.
//   - Filtered element: pkg:$?var=services[name:foo].port — sets port on the named service.
//   - Module root: pkg:$#$ or pkg:json$#config.json — replaces the entire JSON document.
//
// The mutation is in-memory. Call SaveToFile or Mineralize to persist it.
func (mycelium *Mycelium) Inoculate(path string, value any) error {
	target, hypha, err := mycelium.resolveColonyForMutation(path)
	if err != nil {
		return err
	}
	if target != mycelium {
		return target.Inoculate(path, value)
	}
	return target.inoculateLocal(hypha, value)
}

// Graft appends item to the array at path.
//
// The path must point to an array field (no scalar filter on the final segment).
// Wildcards in the module URL are resolved from this mycelium; when the path
// names a different module the mutation is delegated to that colony.
// Example:
//
//	mycelium.Graft("pkg:$?var=services", newServiceMap)
//	mycelium.Graft("pkg:$?var=services[name:foo].handlers[category:main].outbounds", newOutbound)
//
// The mutation is in-memory. Call SaveToFile or Mineralize to persist it.
func (mycelium *Mycelium) Graft(path string, item any) error {
	target, hypha, err := mycelium.resolveColonyForMutation(path)
	if err != nil {
		return err
	}
	if target != mycelium {
		return target.Graft(path, item)
	}
	return target.graftLocal(hypha, item)
}

// Prune removes all items matching path from their parent array.
//
// The final path segment must include a scalar filter (key:value) to identify
// the elements to remove. Without a filter the entire array is cleared.
// Wildcards in the module URL are resolved from this mycelium; when the path
// names a different module the mutation is delegated to that colony.
// Example:
//
//	mycelium.Prune("pkg:$?var=services[name:foo]")
//	mycelium.Prune("pkg:$?var=services[name:foo].handlers[category:main].outbounds[name:bar]")
//
// The mutation is in-memory. Call SaveToFile or Mineralize to persist it.
func (mycelium *Mycelium) Prune(path string) error {
	target, hypha, err := mycelium.resolveColonyForMutation(path)
	if err != nil {
		return err
	}
	if target != mycelium {
		return target.Prune(path)
	}
	return target.pruneLocal(hypha)
}

func (mycelium *Mycelium) inoculateLocal(hypha mushroom.Hypha, value any) error {
	if len(hypha.ResourcePath.Segments) == 0 {
		decoded, err := decodeMutationValue(value)
		if err != nil {
			return err
		}
		mycelium.data = decoded
		return nil
	}

	// Before overwriting, restore any mushroom dereference strings that the caller
	// received as resolved values (via Fruit) but did not intentionally change.
	// This prevents round-trip writes from permanently baking resolved env-var values
	// (e.g. "*pkg:os/env?var=API_KEY") into the JSON file.
	if valueMap, ok := value.(map[string]any); ok {
		if rawCurrent, err := lookup(mycelium.data, hypha.ResourcePath); err == nil {
			raw := rawCurrent
			// Filtered paths (e.g. services[name:ai]) return []any; unwrap the single element.
			if items, ok := rawCurrent.([]any); ok && len(items) == 1 {
				raw = items[0]
			}
			if fruited, err := mycelium.Fruit(raw); err == nil {
				inoculateRestoreDerefs(valueMap, raw, fruited)
			}
		}
	}

	parent, last, err := mycelium.navigateToParent(hypha.ResourcePath.Segments)
	if err != nil {
		return err
	}
	return applySetToParent(parent, last, value)
}

// inoculateRestoreDerefs recursively restores mushroom dereference strings in dst
// that the caller did not intentionally change.
//
// spored is the raw current value (dereference strings intact).
// fruited is the Fruit-resolved version of spored.
//
// For each field in spored whose value is a dereference string (starts with '*'):
//   - If dst[k] equals fruited[k] the caller did not change the field — restore the
//     dereference string so it is not permanently baked into the JSON file.
//   - If dst[k] differs from fruited[k] the caller intentionally wrote a new value —
//     leave dst[k] as-is.
func inoculateRestoreDerefs(dst map[string]any, spored any, fruited any) {
	sporedMap, ok := spored.(map[string]any)
	if !ok {
		return
	}
	fruitedMap, _ := fruited.(map[string]any)

	for k, sporedVal := range sporedMap {
		dstVal, exists := dst[k]
		if !exists {
			continue
		}
		var fruitedVal any
		if fruitedMap != nil {
			fruitedVal = fruitedMap[k]
		}
		dst[k] = inoculateRestoreDerefValue(dstVal, sporedVal, fruitedVal)
	}
}

// inoculateRestoreDerefValue returns the value to store for a single field.
func inoculateRestoreDerefValue(dst any, spored any, fruited any) any {
	switch sv := spored.(type) {
	case string:
		if len(sv) > 0 && sv[0] == '*' && fmt.Sprintf("%v", dst) == fmt.Sprintf("%v", fruited) {
			return sv
		}
	case map[string]any:
		dstMap, ok := dst.(map[string]any)
		if !ok {
			return dst
		}
		inoculateRestoreDerefs(dstMap, sv, fruited)
	case []any:
		dstSlice, ok := dst.([]any)
		if !ok {
			return dst
		}
		fruitedSlice, _ := fruited.([]any)
		for i, sporedElem := range sv {
			if i >= len(dstSlice) {
				break
			}
			var fruitedElem any
			if i < len(fruitedSlice) {
				fruitedElem = fruitedSlice[i]
			}
			dstSlice[i] = inoculateRestoreDerefValue(dstSlice[i], sporedElem, fruitedElem)
		}
	}
	return dst
}

func decodeMutationValue(value any) (any, error) {
	if raw, ok := value.(string); ok {
		return decode(raw)
	}
	return value, nil
}

func (mycelium *Mycelium) graftLocal(hypha mushroom.Hypha, item any) error {
	if len(hypha.ResourcePath.Segments) == 0 {
		return fmt.Errorf("json substrate: empty resource path")
	}
	parent, last, err := mycelium.navigateToParent(hypha.ResourcePath.Segments)
	if err != nil {
		return err
	}
	return applyGraftToParent(parent, last, item)
}

func (mycelium *Mycelium) pruneLocal(hypha mushroom.Hypha) error {
	if len(hypha.ResourcePath.Segments) == 0 {
		return fmt.Errorf("json substrate: empty resource path")
	}
	parent, last, err := mycelium.navigateToParent(hypha.ResourcePath.Segments)
	if err != nil {
		return err
	}
	return applyPruneToParent(parent, last)
}

// resolveColonyForMutation resolves path into the colony that owns the target
// module, germinating unknown JSON modules on demand (same routing as Spore).
func (mycelium *Mycelium) resolveColonyForMutation(path string) (*Mycelium, mushroom.Hypha, error) {
	var hypha mushroom.Hypha
	for {
		var err error
		hypha, err = mycelium.soil.Hypha(path, mycelium.url)
		if err == nil {
			break
		}
		var unrecognized *mushroom.ErrUnrecognizedMycelium
		if !errors.As(err, &unrecognized) {
			return nil, mushroom.Hypha{}, err
		}
		if _, germinateErr := mycelium.soil.Germinate(unrecognized.Hypha, unrecognized.Substrate); germinateErr != nil {
			return nil, mushroom.Hypha{}, fmt.Errorf("json substrate: %q: %w", path, germinateErr)
		}
	}

	if !hypha.URL {
		return mycelium, hypha, nil
	}

	colony, substrate, recognizeErr := mycelium.soil.Recognize(hypha)
	if recognizeErr != nil {
		return nil, hypha, fmt.Errorf("json substrate: %q: %w", path, recognizeErr)
	}

	if colony != nil {
		target, ok := colony.(*Mycelium)
		if !ok {
			return nil, hypha, fmt.Errorf("json substrate: %q: colony is %T, want *json_substrate.Mycelium", path, colony)
		}
		return target, hypha, nil
	}

	m, germinateErr := mycelium.soil.Germinate(hypha, substrate)
	if germinateErr != nil {
		return nil, hypha, fmt.Errorf("json substrate: %q: %w", path, germinateErr)
	}
	target, ok := m.(*Mycelium)
	if !ok {
		return nil, hypha, fmt.Errorf("json substrate: %q: germinated colony is %T, want *json_substrate.Mycelium", path, m)
	}
	return target, hypha, nil
}

func (mycelium *Mycelium) MushroomURL() string {
	return mycelium.url.String()
}

func (mycelium *Mycelium) MyceliumURL() mushroom.Hypha {
	return mycelium.url
}

func (mycelium *Mycelium) Soil() *mushroom.Soil {
	return mycelium.soil
}

func (mycelium *Mycelium) Substrate() *mushroom.Substrate {
	return &mycelium.substrate
}

func decode(data any) (any, error) {
	raw, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("json substrate: unsupported digest data %T", data)
	}

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}

	return decoded, nil
}

func lookup(data any, path mushroom.ResourcePath) (any, error) {
	if len(path.Segments) == 0 {
		return nil, fmt.Errorf("json substrate: empty resource path")
	}

	current := data
	for _, segment := range path.Segments {
		// A segment with a Call is a built-in function (first, last) applied to
		// the current value. Handle it before the array and object branches so
		// it works regardless of the upstream context.
		if segment.Call != nil {
			var err error
			current, err = applyBuiltinCall(current, *segment.Call)
			if err != nil {
				return nil, err
			}
			continue
		}

		if items, ok := current.([]any); ok {
			var err error
			current, err = lookupArraySegment(items, segment)
			if err != nil {
				return nil, err
			}
			continue
		}

		if segment.Name != "" {
			object, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("json substrate: %q is not an object", segment.Name)
			}

			var exists bool
			current, exists = object[segment.Name]
			if !exists {
				return nil, fmt.Errorf("json substrate: key %q not found", segment.Name)
			}
		}

		for _, scalar := range segment.Scalars {
			var err error
			current, err = applyScalar(current, scalar)
			if err != nil {
				return nil, err
			}
		}
	}

	return current, nil
}

// applyBuiltinCall evaluates a built-in path call (first, last) on the current value.
func applyBuiltinCall(current any, call mushroom.ResourceCall) (any, error) {
	items, ok := current.([]any)
	if !ok {
		return nil, fmt.Errorf("json substrate: %q() requires an array", call.Name)
	}
	switch call.Name {
	case "first":
		if len(items) == 0 {
			return nil, fmt.Errorf("json substrate: first() on empty array")
		}
		return items[0], nil
	case "last":
		if len(items) == 0 {
			return nil, fmt.Errorf("json substrate: last() on empty array")
		}
		return items[len(items)-1], nil
	default:
		return nil, fmt.Errorf("json substrate: unknown call %q", call.Name)
	}
}

func lookupArraySegment(items []any, segment mushroom.ResourcePathSegment) (any, error) {
	// Primary attempt: filter items by scalars, then return the named field from
	// the first matching item. This handles paths like name[key] where the scalars
	// filter the current array items.
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if !matchesScalars(object, segment.Scalars) {
			continue
		}
		if segment.Name == "" {
			return object, nil
		}
		if value, ok := object[segment.Name]; ok {
			return value, nil
		}
	}

	// Fallback: navigate into the named sub-array of each item and apply the
	// scalars as a filter there. This handles paths like
	// services[name:hello-world].handlers[category:main] where
	// services[name:hello-world] returns [hello-world-service] and the next
	// segment must drill into service["handlers"] and filter by category=main.
	if segment.Name != "" && len(segment.Scalars) > 0 {
		var matches []any
		for _, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			sub, ok := object[segment.Name]
			if !ok {
				continue
			}
			subItems, ok := sub.([]any)
			if !ok {
				continue
			}
			for _, subItem := range subItems {
				subObj, ok := subItem.(map[string]any)
				if !ok {
					continue
				}
				if matchesScalars(subObj, segment.Scalars) {
					matches = append(matches, subObj)
				}
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return matches, nil
		}
	}

	return nil, fmt.Errorf("json substrate: array segment %q not found", segment.Name)
}

func applyScalar(current any, scalar mushroom.ResourceScalar) (any, error) {
	switch scalar.Kind {
	case mushroom.ResourceScalarCall:
		// Scalar calls use $.builtin() syntax where $ refers to the current value.
		// For example services[$.first()] and services[type:Proxy][$.last()].
		if scalar.Call == nil {
			return nil, fmt.Errorf("json substrate: scalar call is nil")
		}
		name := strings.TrimPrefix(scalar.Call.Name, "$.")
		if name == scalar.Call.Name {
			return nil, fmt.Errorf("json substrate: scalar call %q must start with $.", scalar.Call.Name)
		}
		return applyBuiltinCall(current, mushroom.ResourceCall{Name: name, Args: scalar.Call.Args})
	case mushroom.ResourceScalarNumber:
		index, err := strconv.Atoi(scalar.Value)
		if err != nil {
			return nil, fmt.Errorf("json substrate: invalid array index %q", scalar.Value)
		}

		items, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("json substrate: index %d requires an array", index)
		}
		if index < 0 || index >= len(items) {
			return nil, fmt.Errorf("json substrate: index %d out of range", index)
		}

		return items[index], nil
	case mushroom.ResourceScalarKey:
		items, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("json substrate: key scalar requires an array")
		}

		for _, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if value, ok := object[scalar.Key]; ok {
				return value, nil
			}
		}

		return nil, fmt.Errorf("json substrate: key %q not found in array", scalar.Key)
	case mushroom.ResourceScalarKeyValue:
		items, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("json substrate: key-value scalar requires an array")
		}

		var matches []any
		for _, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value, ok := object[scalar.Key]
			if !ok {
				continue
			}
			if scalar.Value == "$" || fmt.Sprint(value) == scalar.Value {
				matches = append(matches, object)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("json substrate: key-value %q:%q not found in array", scalar.Key, scalar.Value)
		}
		return matches, nil
	default:
		return nil, fmt.Errorf("json substrate: unsupported scalar kind %q", scalar.Kind)
	}
}

// navigateToParent returns the direct parent container and the last segment.
// For a single-segment path the parent is mycelium.data itself.
func (mycelium *Mycelium) navigateToParent(segs []mushroom.ResourcePathSegment) (any, mushroom.ResourcePathSegment, error) {
	if len(segs) == 1 {
		return mycelium.data, segs[0], nil
	}
	parentPath := mushroom.ResourcePath{Segments: segs[:len(segs)-1]}
	parent, err := lookup(mycelium.data, parentPath)
	if err != nil {
		return nil, mushroom.ResourcePathSegment{}, fmt.Errorf("json substrate: parent path: %w", err)
	}
	return parent, segs[len(segs)-1], nil
}

// applySetToParent sets value at the location identified by seg within parent.
// parent may be a map[string]any or a []any (from a prior filter returning
// multiple matches); in the latter case the set is applied to every map element.
func applySetToParent(parent any, seg mushroom.ResourcePathSegment, value any) error {
	switch p := parent.(type) {
	case map[string]any:
		return setOnMap(p, seg, value)
	case []any:
		found := false
		for _, item := range p {
			if obj, ok := item.(map[string]any); ok {
				if err := setOnMap(obj, seg, value); err == nil {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("json substrate: no settable element in parent array for %q", seg.Name)
		}
		return nil
	default:
		return fmt.Errorf("json substrate: expected object or array as parent for inoculate, got %T", parent)
	}
}

// setOnMap performs the actual write on a map parent.
// For a name-only segment it sets the field directly.
// For a name + scalar it navigates into the named array and replaces the element
// identified by the scalar.
func setOnMap(parent map[string]any, seg mushroom.ResourcePathSegment, value any) error {
	if seg.Name == "" {
		return fmt.Errorf("json substrate: segment name required for inoculate")
	}
	if len(seg.Scalars) == 0 {
		parent[seg.Name] = value
		return nil
	}
	if len(seg.Scalars) != 1 {
		return fmt.Errorf("json substrate: inoculate supports exactly one scalar on the last segment, got %d", len(seg.Scalars))
	}
	field, ok := parent[seg.Name]
	if !ok {
		return fmt.Errorf("json substrate: field %q not found", seg.Name)
	}
	items, ok := field.([]any)
	if !ok {
		return fmt.Errorf("json substrate: field %q must be an array for indexed inoculate (got %T)", seg.Name, field)
	}
	return setByScalar(items, seg.Name, seg.Scalars[0], value)
}

// setByScalar replaces the element(s) in items that match scalar with value.
func setByScalar(items []any, fieldName string, scalar mushroom.ResourceScalar, value any) error {
	switch scalar.Kind {
	case mushroom.ResourceScalarNumber:
		idx, err := strconv.Atoi(scalar.Value)
		if err != nil {
			return fmt.Errorf("json substrate: invalid index %q", scalar.Value)
		}
		if idx < 0 || idx >= len(items) {
			return fmt.Errorf("json substrate: index %d out of range (len %d)", idx, len(items))
		}
		items[idx] = value
		return nil
	case mushroom.ResourceScalarKeyValue:
		found := false
		for i, item := range items {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			v, ok := obj[scalar.Key]
			if !ok {
				continue
			}
			if fmt.Sprint(v) == scalar.Value {
				items[i] = value
				found = true
			}
		}
		if !found {
			return fmt.Errorf("json substrate: %q:%q not found in field %q", scalar.Key, scalar.Value, fieldName)
		}
		return nil
	default:
		return fmt.Errorf("json substrate: unsupported scalar kind %q for inoculate", scalar.Kind)
	}
}

// applyGraftToParent appends item to the array identified by seg within parent.
func applyGraftToParent(parent any, seg mushroom.ResourcePathSegment, item any) error {
	switch p := parent.(type) {
	case map[string]any:
		return graftOnMap(p, seg, item)
	case []any:
		found := false
		for _, elem := range p {
			if obj, ok := elem.(map[string]any); ok {
				if err := graftOnMap(obj, seg, item); err == nil {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("json substrate: no graftable element in parent array for %q", seg.Name)
		}
		return nil
	default:
		return fmt.Errorf("json substrate: expected object or array as parent for graft, got %T", parent)
	}
}

// graftOnMap appends item to the array at parent[seg.Name].
func graftOnMap(parent map[string]any, seg mushroom.ResourcePathSegment, item any) error {
	if seg.Name == "" {
		return fmt.Errorf("json substrate: segment name required for graft")
	}
	if len(seg.Scalars) > 0 {
		return fmt.Errorf("json substrate: graft target segment must not have scalars")
	}
	var items []any
	if existing := parent[seg.Name]; existing != nil {
		var ok bool
		items, ok = existing.([]any)
		if !ok {
			return fmt.Errorf("json substrate: field %q must be an array for graft (got %T)", seg.Name, existing)
		}
	}
	parent[seg.Name] = append(items, item)
	return nil
}

// applyPruneToParent removes elements matching seg from the array at seg.Name within parent.
func applyPruneToParent(parent any, seg mushroom.ResourcePathSegment) error {
	switch p := parent.(type) {
	case map[string]any:
		return pruneOnMap(p, seg)
	case []any:
		found := false
		for _, elem := range p {
			if obj, ok := elem.(map[string]any); ok {
				if err := pruneOnMap(obj, seg); err == nil {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("json substrate: no prunable element in parent array for %q", seg.Name)
		}
		return nil
	default:
		return fmt.Errorf("json substrate: expected object or array as parent for prune, got %T", parent)
	}
}

// pruneOnMap removes elements from parent[seg.Name] that match seg.Scalars.
// If there are no scalars the entire array is cleared.
func pruneOnMap(parent map[string]any, seg mushroom.ResourcePathSegment) error {
	if seg.Name == "" {
		return fmt.Errorf("json substrate: segment name required for prune")
	}
	existing, ok := parent[seg.Name]
	if !ok {
		return fmt.Errorf("json substrate: field %q not found", seg.Name)
	}
	items, ok := existing.([]any)
	if !ok {
		return fmt.Errorf("json substrate: field %q must be an array for prune (got %T)", seg.Name, existing)
	}
	if len(seg.Scalars) == 0 {
		parent[seg.Name] = []any{}
		return nil
	}
	var kept []any
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			kept = append(kept, item) // non-object elements are always kept
			continue
		}
		if !matchesScalars(obj, seg.Scalars) {
			kept = append(kept, item)
		}
	}
	parent[seg.Name] = kept
	return nil
}

func matchesScalars(object map[string]any, scalars []mushroom.ResourceScalar) bool {
	for _, scalar := range scalars {
		switch scalar.Kind {
		case mushroom.ResourceScalarKey:
			if _, ok := object[scalar.Key]; !ok {
				return false
			}
		case mushroom.ResourceScalarKeyValue:
			value, ok := object[scalar.Key]
			if !ok {
				return false
			}
			if scalar.Value != "$" && fmt.Sprint(value) != scalar.Value {
				return false
			}
		case mushroom.ResourceScalarNumber:
			return false
		}
	}

	return true
}
