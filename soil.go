package mushroom

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Soil struct {
	substrates []Substrate
	colonies   []Mycelium
}

var (
	ErrUnrecognizedHypha = errors.New("mushroom: hypha is not recognized")
	ErrInvalidSubstrate  = errors.New("mushroom: substrate MushroomURL must be a link")
)

// ErrUnrecognizedMycelium is returned by Hypha (via evalLambda) when a
// dereference lambda has a matching substrate but no loaded colony. Substrate
// is the substrate that can materialise the mycelium; callers (e.g. Spore)
// should Forage+Digest it and retry.
type ErrUnrecognizedMycelium struct {
	Hypha     Hypha
	Substrate Substrate
}

func (e *ErrUnrecognizedMycelium) Error() string {
	return fmt.Sprintf("mushroom: mycelium %q is not loaded", e.Hypha.String())
}

// ErrUnrecognizedSubstrate is returned by Hypha (via evalLambda) when a
// dereference lambda's URL is not known to any colony or substrate in the soil.
type ErrUnrecognizedSubstrate struct {
	Hypha Hypha
}

func (e *ErrUnrecognizedSubstrate) Error() string {
	return fmt.Sprintf("mushroom: no substrate found for %q", e.Hypha.String())
}

func (soil *Soil) AddSubstrate(substrate Substrate) error {
	if substrate == nil {
		return ErrInvalidSubstrate
	}

	hypha, err := soil.Hypha(substrate.MushroomURL())
	if err != nil || !hypha.URL || hypha.Dereference {
		return ErrInvalidSubstrate
	}

	soil.substrates = append(soil.substrates, substrate)
	return nil
}

func (soil *Soil) AddColony(mycelium Mycelium) {
	if mycelium == nil {
		return
	}

	soil.colonies = append(soil.colonies, mycelium)
}

// Germinate forages raw data for hypha from substrate, digests it into a new
// Mycelium, and registers both the mycelium (as a colony) and the substrate
// (if not already present) into the soil. It is the canonical way to load an
// unrecognised mycelium on demand.
func (soil *Soil) Germinate(hypha Hypha, substrate Substrate) (Mycelium, error) {
	alreadyRegistered := false
	for _, s := range soil.substrates {
		if s.MushroomURL() == substrate.MushroomURL() {
			alreadyRegistered = true
			break
		}
	}
	if !alreadyRegistered {
		if err := soil.AddSubstrate(substrate); err != nil {
			return nil, err
		}
	}

	moduleHypha := hypha.ModuleURL()
	data, err := substrate.Forage(moduleHypha)
	if err != nil {
		return nil, fmt.Errorf("mushroom: germinate %q: forage: %w", hypha.String(), err)
	}
	m, err := substrate.Digest(moduleHypha, data, soil)
	if err != nil {
		return nil, fmt.Errorf("mushroom: germinate %q: digest: %w", hypha.String(), err)
	}
	soil.AddColony(m)
	return m, nil
}

func (soil *Soil) Substrates() []Substrate {
	return append([]Substrate(nil), soil.substrates...)
}

func (soil *Soil) Colony() []Mycelium {
	return append([]Mycelium(nil), soil.colonies...)
}

// Recognize returns the colony or substrate that matches hypha.
// The return values behave like a radio group: exactly one of Mycelium or
// Substrate will be non-nil on success; the other is always nil. On failure
// both are nil and error is non-nil.
func (soil *Soil) Recognize(hypha Hypha) (Mycelium, Substrate, error) {
	for _, mycelium := range soil.colonies {
		mh, err := soil.Hypha(mycelium.MushroomURL())
		if err == nil && mh.Satisfies(hypha) {
			return mycelium, nil, nil
		}
	}

	for _, substrate := range soil.substrates {
		sh, err := soil.Hypha(substrate.MushroomURL())
		if err == nil && sh.Satisfies(hypha) {
			return nil, substrate, nil
		}
	}

	return nil, nil, ErrUnrecognizedHypha
}

type DereferenceType string

const (
	DereferenceTypeModule   DereferenceType = "module"
	DereferenceTypeResource DereferenceType = "resource"
)

type ResourceKind string

const (
	ResourceKindVar  ResourceKind = "var"
	ResourceKindFunc ResourceKind = "func"
	ResourceKindObj  ResourceKind = "obj"
)

type ResourcePath struct {
	Raw      string
	Segments []ResourcePathSegment
}

func (path ResourcePath) String() string {
	return path.Raw
}

// ParentResource returns a copy of h with the parent resource path, with the
// last path step removed. Link and dereference flags are preserved.
// See ParentResourceURL for ok semantics.
func (h Hypha) ParentResource() (Hypha, bool) {
	if !h.URL {
		return h, false
	}
	if h.ResourceKind == "" || len(h.ResourcePath.Segments) == 0 {
		return h, false
	}

	segs := h.ResourcePath.Segments
	var parentPath ResourcePath
	switch len(segs) {
	case 1:
		if len(segs[0].Scalars) == 0 {
			return h, false
		}
		parentPath = resourcePathFromSegments([]ResourcePathSegment{{Name: segs[0].Name}})
	default:
		last := segs[len(segs)-1]
		if len(last.Scalars) > 0 {
			parentPath = resourcePathFromSegments(append(append([]ResourcePathSegment(nil), segs[:len(segs)-1]...), ResourcePathSegment{Name: last.Name}))
		} else {
			parentPath = resourcePathFromSegments(segs[:len(segs)-1])
		}
	}

	result := h
	result.ResourcePath = parentPath
	result.Path = replaceResourcePath(h.Path, h.ResourceKind, h.ResourcePath.Raw, parentPath.Raw)
	return result, true
}

// ParentResourceURL returns the parent Mushroom URL string of h's resource path,
// with the last path step removed.
//
// Symbolic paths (no pkg: prefix) are returned unchanged with ok false.
// URLs without a resource path, or with a single plain segment (no [scalar]),
// are returned unchanged with ok false.
// A single segment with [scalar] selectors drops the selectors.
// When the last segment has [scalar] selectors, only those selectors are removed.
// Otherwise the last dot-separated segment is dropped.
func (h Hypha) ParentResourceURL() (string, bool) {
	parent, ok := h.ParentResource()
	if !ok {
		if !h.URL {
			return h.Path, false
		}
		return h.String(), false
	}
	return parent.String(), true
}

// ParentResourceURL parses mushroomURL and returns its parent resource URL.
func ParentResourceURL(mushroomURL string) (string, bool) {
	hypha, err := (&Soil{}).Hypha(mushroomURL)
	if err != nil {
		return mushroomURL, false
	}
	return hypha.ParentResourceURL()
}

func resourceSegmentHasData(seg ResourcePathSegment) bool {
	return len(seg.Scalars) > 0 || seg.Call != nil
}

// ChildResource extends h's resource path with child.
//
// When child is a scalar (starts with '[' and ends with ']'), it is appended to
// the last segment's selectors. The last segment must not already have selectors
// or a call; otherwise an error is returned.
//
// When child is not a scalar, it is parsed as a new dot-separated segment and
// appended to the path. All spaces in child are trimmed.
func (h Hypha) ChildResource(child string) (Hypha, error) {
	child = stripIgnoredRunes(strings.TrimSpace(child))
	if child == "" {
		return Hypha{}, fmt.Errorf("mushroom: child resource is empty")
	}
	if !h.URL {
		return Hypha{}, fmt.Errorf("mushroom: child resource requires a mushroom URL")
	}
	if h.ResourceKind == "" || len(h.ResourcePath.Segments) == 0 {
		return Hypha{}, fmt.Errorf("mushroom: child resource requires a resource path")
	}

	var nextPath ResourcePath
	if strings.HasPrefix(child, "[") && strings.HasSuffix(child, "]") {
		scalar, ok := parseResourceScalar(stripIgnoredRunes(child[1 : len(child)-1]))
		if !ok {
			return Hypha{}, fmt.Errorf("mushroom: invalid child scalar %q", child)
		}

		segs := append([]ResourcePathSegment(nil), h.ResourcePath.Segments...)
		last := segs[len(segs)-1]
		if resourceSegmentHasData(last) {
			return Hypha{}, fmt.Errorf("mushroom: last segment %q already has selectors", last.Name)
		}
		last.Scalars = append(last.Scalars, scalar)
		segs[len(segs)-1] = last
		nextPath = resourcePathFromSegments(segs)
	} else {
		segment, ok := parseResourcePathSegment(child)
		if !ok {
			return Hypha{}, fmt.Errorf("mushroom: invalid child segment %q", child)
		}
		nextPath = resourcePathFromSegments(append(append([]ResourcePathSegment(nil), h.ResourcePath.Segments...), segment))
	}

	result := h
	result.ResourcePath = nextPath
	result.Path = replaceResourcePath(h.Path, h.ResourceKind, h.ResourcePath.Raw, nextPath.Raw)
	return result, nil
}

func replaceResourcePath(path string, kind ResourceKind, oldRaw, newRaw string) string {
	marker := "?" + string(kind) + "=" + oldRaw
	if idx := strings.Index(path, marker); idx != -1 {
		return path[:idx] + "?" + string(kind) + "=" + newRaw
	}

	normalized := stripIgnoredRunes(path)
	if idx := strings.Index(normalized, marker); idx != -1 {
		return normalized[:idx] + "?" + string(kind) + "=" + newRaw
	}

	return path
}

func resourcePathFromSegments(segs []ResourcePathSegment) ResourcePath {
	parts := make([]string, len(segs))
	for i, seg := range segs {
		parts[i] = resourceSegmentString(seg)
	}
	return ResourcePath{
		Raw:      strings.Join(parts, "."),
		Segments: append([]ResourcePathSegment(nil), segs...),
	}
}

func resourceSegmentString(seg ResourcePathSegment) string {
	var builder strings.Builder
	builder.WriteString(seg.Name)
	for _, scalar := range seg.Scalars {
		builder.WriteByte('[')
		builder.WriteString(resourceScalarString(scalar))
		builder.WriteByte(']')
	}
	if seg.Call != nil {
		builder.WriteByte('(')
		builder.WriteString(resourceCallArgsString(seg.Call.Args))
		builder.WriteByte(')')
	}
	return builder.String()
}

func resourceScalarString(scalar ResourceScalar) string {
	switch scalar.Kind {
	case ResourceScalarKeyValue:
		return scalar.Key + ":" + scalar.Value
	case ResourceScalarNumber:
		return scalar.Value
	case ResourceScalarKey:
		return scalar.Key
	case ResourceScalarCall:
		if scalar.Call == nil {
			return ""
		}
		return scalar.Call.Name + "(" + resourceCallArgsString(scalar.Call.Args) + ")"
	default:
		return ""
	}
}

func resourceCallArgsString(args []ResourceScalar) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = resourceScalarString(arg)
	}
	return strings.Join(parts, ",")
}

type ResourcePathSegment struct {
	Name    string
	Scalars []ResourceScalar
	Call    *ResourceCall
}

type ResourceScalarKind string

const (
	ResourceScalarKeyValue ResourceScalarKind = "key_value"
	ResourceScalarNumber   ResourceScalarKind = "number"
	ResourceScalarKey      ResourceScalarKind = "key"
	ResourceScalarCall     ResourceScalarKind = "call"
)

type ResourceScalar struct {
	Kind  ResourceScalarKind
	Key   string
	Value string

	// Call is set when Kind is ResourceScalarCall.
	// It holds an inline function call that appears in a scalar position,
	// e.g. $.first() or $.last() inside path[$.first()].
	// This is distinct from ResourcePathSegment.Call, which marks a segment
	// that is itself the evaluated function for a func= resource.
	Call *ResourceCall
}

type ResourceCall struct {
	Name string
	Args []ResourceScalar
}

type Hypha struct {
	Path            string // The original path: "some random data" | "pkg:$?var=services[0]"
	Type            string // Package type same as package-url
	Dereference     bool   // Whether the path is a dereference: true | false
	DereferenceType DereferenceType
	PackageID       string // Package url with its namespace and name
	ModuleID        string // Module name or file name
	ResourceKind    ResourceKind
	ResourcePath    ResourcePath // Resource path such as path0, path0.path1, or path[scalar]
	AdditionalProps map[string]string
	URL             bool // if path has `pkg:` prefixed its a mushroom url. Otherwise its just a symbol
}

// AsLink returns a copy of the hypha with the dereference flag cleared.
// If the hypha is not a URL it is returned unchanged.
func (h Hypha) AsLink() Hypha {
	if !h.URL {
		return h
	}
	h.Dereference = false
	h.DereferenceType = ""
	return h
}

// AsDereference returns a copy of the hypha with the dereference flag set.
// If no DereferenceType is supplied it defaults to DereferenceTypeResource.
// If the hypha is not a URL it is returned unchanged.
func (h Hypha) AsDereference(dt ...DereferenceType) Hypha {
	if !h.URL {
		return h
	}
	h.Dereference = true
	if len(dt) > 0 {
		h.DereferenceType = dt[0]
	} else {
		h.DereferenceType = DereferenceTypeResource
	}
	return h
}

// ModuleURL returns a copy of the hypha stripped to its module-level identity:
// URL, Type, PackageID, ModuleID, and AdditionalProps only. Dereference,
// ResourceKind, and ResourcePath are cleared. If the hypha is not a URL it is
// returned unchanged.
func (h Hypha) ModuleURL() Hypha {
	if !h.URL {
		return h
	}
	return Hypha{
		URL:             true,
		Type:            h.Type,
		PackageID:       h.PackageID,
		ModuleID:        h.ModuleID,
		AdditionalProps: h.AdditionalProps,
	}
}

// Hypha parses path into a structured Hypha, recursively evaluating any lambda
// expressions (…) before structural parsing. Lambda execution is driven by the
// soil's registered colonies.
//
// An optional defaults Hypha can be provided; any empty or wildcard ($) field
// in the parsed result is filled from defaults, provided defaults contains no
// function-call segments in its resource path.
//
// Returns an error when a lambda cannot be resolved (e.g. a dereference lambda
// with no matching colony).
func (soil *Soil) Hypha(path string, defaults ...Hypha) (Hypha, error) {
	hypha := Hypha{Path: path}

	resolved, err := soil.resolvePathLambdas(path, defaults...)
	if err != nil {
		return hypha, err
	}

	normalized := stripIgnoredRunes(resolved)

	switch {
	case strings.HasPrefix(normalized, "*pkg:"):
		hypha.URL = true
		hypha.Dereference = true
		hypha.DereferenceType = DereferenceTypeResource
		normalized = strings.TrimPrefix(normalized, "*pkg:")
	case strings.HasPrefix(normalized, "pkg:"):
		hypha.URL = true
		normalized = strings.TrimPrefix(normalized, "pkg:")
	default:
		return hypha, nil
	}

	packageAndSections := parseTypeAndPackage(normalized, &hypha)
	parseSections(packageAndSections, &hypha)

	if len(defaults) > 0 {
		fillHyphaFromDefault(&hypha, defaults[0])
	}

	return hypha, nil
}

// resolvePathLambdas replaces every lambda expression (…) in raw with its
// evaluated string result and returns the fully concrete path.
// A ( is a lambda start when NOT immediately preceded by an identifier character
// (letter, digit, _ or -); otherwise it is a function-call parenthesis copied
// verbatim. Resolution is recursive: the result is re-processed until no more
// lambdas remain.
func (soil *Soil) resolvePathLambdas(raw string, defaults ...Hypha) (string, error) {
	var result strings.Builder
	remaining := raw
	for remaining != "" {
		idx := strings.IndexByte(remaining, '(')
		if idx == -1 {
			result.WriteString(remaining)
			break
		}
		prefix := remaining[:idx]
		if !isLambdaPosition(result.String(), prefix) {
			// Function-call parenthesis — copy through verbatim.
			result.WriteString(prefix)
			result.WriteByte('(')
			remaining = remaining[idx+1:]
			continue
		}
		result.WriteString(prefix)
		remaining = remaining[idx:]
		content, rest, ok := takeBalanced(remaining, '(', ')')
		if !ok {
			return "", fmt.Errorf("mushroom: unbalanced parentheses in path %q", raw)
		}
		v, err := soil.evalLambda(content, defaults...)
		if err != nil {
			return "", fmt.Errorf("mushroom: lambda %q: %w", content, err)
		}
		result.WriteString(v)
		remaining = rest
	}

	resolved := result.String()
	if pathContainsLambda(resolved) && resolved != raw {
		return soil.resolvePathLambdas(resolved, defaults...)
	}
	return resolved, nil
}

// evalLambda evaluates the content of one lambda expression and returns its
// string representation.
//
//   - Symbol (no pkg: prefix): returned verbatim as a plain string.
//   - Non-dereference URL: if a matching colony is found its Link is called;
//     otherwise the resolved link string is returned as-is.
//   - Dereference URL: a matching colony must exist; its Spore result is
//     formatted with fmt.Sprint and returned. Returns an error when no colony
//     is found.
func (soil *Soil) evalLambda(content string, defaults ...Hypha) (string, error) {
	h, err := soil.Hypha(content, defaults...)
	if err != nil {
		return "", err
	}

	if !h.URL {
		return content, nil
	}

	if !h.Dereference {
		return h.String(), nil
	}

	colony, substrate, err := soil.Recognize(h)
	if err != nil {
		return "", &ErrUnrecognizedSubstrate{Hypha: h}
	}
	if colony == nil {
		return "", &ErrUnrecognizedMycelium{Hypha: h, Substrate: substrate}
	}

	val, err := colony.Spore(h.String())
	if err != nil {
		return "", err
	}
	return fmt.Sprint(val), nil
}

// isLambdaPosition reports whether the ( that follows the already-written text
// (already) and the literal prefix before the ( is a lambda start rather than
// a function-call parenthesis.
// A ( is a lambda when not immediately preceded by an identifier character.
func isLambdaPosition(already, prefix string) bool {
	full := already + prefix
	if full == "" {
		return true
	}
	return !isIdentChar(full[len(full)-1])
}

// fillHyphaFromDefault copies concrete fields from def into hypha for any field
// that is empty or a bare wildcard ($), but only if hypha contains at least one
// explicit $ wildcard in Type, PackageID, or ModuleID.
// A URL with no $ at all is taken literally and is never filled.
// def must have no lambda scalars and no call segments in its resource path.
func fillHyphaFromDefault(hypha *Hypha, def Hypha) {
	hasWildcard := hypha.Type == "$" || hypha.PackageID == "$" || hypha.ModuleID == "$"
	if !hasWildcard {
		return
	}
	if !isConcreteDefault(def) {
		return
	}
	if (hypha.Type == "" || hypha.Type == "$") && def.Type != "" && def.Type != "$" {
		hypha.Type = def.Type
	}
	if (hypha.PackageID == "" || hypha.PackageID == "$") && def.PackageID != "" && def.PackageID != "$" {
		hypha.PackageID = def.PackageID
	}
	if (hypha.ModuleID == "" || hypha.ModuleID == "$") && def.ModuleID != "" && def.ModuleID != "$" {
		hypha.ModuleID = def.ModuleID
	}
}

// isConcreteDefault returns true when h can be safely used as a defaults Hypha:
// no lambda scalars and no call segments in its resource path.
func isConcreteDefault(h Hypha) bool {
	for _, seg := range h.ResourcePath.Segments {
		if seg.Call != nil {
			return false
		}
		for _, s := range seg.Scalars {
			if s.Kind == ResourceScalarCall {
				return false
			}
		}
	}
	return true
}

func (hypha Hypha) String() string {
	if !hypha.URL {
		return hypha.Path
	}

	var builder strings.Builder
	if hypha.Dereference && hypha.DereferenceType == DereferenceTypeResource {
		builder.WriteString("*")
	}
	builder.WriteString("pkg:")
	if hypha.Type != "" {
		builder.WriteString(hypha.Type)
	}

	packageID := hypha.PackageID
	if packageID == "" {
		packageID = "$"
	}
	if hypha.Type != "" && packageID != "$" {
		builder.WriteString("/")
	}
	builder.WriteString(packageID)

	moduleID := hypha.ModuleID
	if moduleID == "" {
		moduleID = "$"
	}
	builder.WriteString("#")
	if hypha.Dereference && hypha.DereferenceType == DereferenceTypeModule {
		builder.WriteString("*")
	}
	builder.WriteString(moduleID)

	resource := hypha.resourceString()
	if resource != "" {
		builder.WriteString("?")
		builder.WriteString(resource)
	}

	return builder.String()
}

func (hypha Hypha) resourceString() string {
	var builder strings.Builder
	if hypha.ResourceKind != "" {
		builder.WriteString(string(hypha.ResourceKind))
		builder.WriteString("=")
		builder.WriteString(hypha.ResourcePath.String())
	}

	props := additionalPropsString(hypha.AdditionalProps)
	if props == "" {
		return builder.String()
	}
	if builder.Len() > 0 {
		builder.WriteString("&")
	}
	builder.WriteString(props)

	return builder.String()
}

func stripIgnoredRunes(path string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}

		return r
	}, path)
}

func parseTypeAndPackage(path string, hypha *Hypha) string {
	if strings.HasPrefix(path, "$") {
		sectionStart := indexAny(path, "#?")
		if sectionStart == -1 {
			hypha.PackageID = path
			return ""
		}

		hypha.PackageID = path[:sectionStart]
		return path[sectionStart:]
	}

	typeEnd := indexAny(path, "/$#?")
	if typeEnd == -1 {
		hypha.Type = path
		return ""
	}

	hypha.Type = path[:typeEnd]
	if path[typeEnd] == '/' {
		packagePath := path[typeEnd+1:]
		sectionStart := indexAny(packagePath, "#?")
		if sectionStart == -1 {
			hypha.PackageID = packagePath
			return ""
		}

		hypha.PackageID = packagePath[:sectionStart]
		return packagePath[sectionStart:]
	}

	sectionStart := indexAny(path[typeEnd:], "#?")
	if sectionStart == -1 {
		hypha.PackageID = path[typeEnd:]
		return ""
	}

	hypha.PackageID = path[typeEnd : typeEnd+sectionStart]
	return path[typeEnd+sectionStart:]
}

func parseSections(path string, hypha *Hypha) {
	if path == "" {
		return
	}

	if strings.HasPrefix(path, "#") {
		moduleAndResource := path[1:]
		resourceStart := strings.Index(moduleAndResource, "?")
		if resourceStart == -1 {
			parseModule(moduleAndResource, hypha)
			return
		}

		parseModule(moduleAndResource[:resourceStart], hypha)
		parseResource(moduleAndResource[resourceStart+1:], hypha)
		return
	}

	if strings.HasPrefix(path, "?") {
		parseResource(path[1:], hypha)
	}
}

func parseModule(moduleID string, hypha *Hypha) {
	if strings.HasPrefix(moduleID, "*") {
		hypha.Dereference = true
		hypha.DereferenceType = DereferenceTypeModule
		hypha.ModuleID = strings.TrimPrefix(moduleID, "*")
		return
	}

	hypha.ModuleID = moduleID
}

func parseResource(resource string, hypha *Hypha) {
	resourcePart, additionalProps, _ := strings.Cut(resource, "&")
	hypha.AdditionalProps = parseAdditionalProps(additionalProps)

	kind, value, ok := strings.Cut(resourcePart, "=")
	if !ok {
		return
	}

	switch kind {
	case string(ResourceKindVar), string(ResourceKindFunc), string(ResourceKindObj):
		resourceKind := ResourceKind(kind)
		resourcePath, ok := parseResourcePath(value, resourceKind)
		if !ok {
			return
		}
		hypha.ResourceKind = resourceKind
		hypha.ResourcePath = resourcePath
	}
}

func parseAdditionalProps(props string) map[string]string {
	if props == "" {
		return nil
	}

	result := make(map[string]string)
	for _, pair := range strings.Split(props, "&") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}

		result[key] = value
	}

	return result
}

// ParseResourcePath parses a concrete (lambda-free) resource path string.
// Substrates call this after resolving all lambdas in a raw path string.
func ParseResourcePath(raw string, kind ResourceKind) (ResourcePath, bool) {
	return parseResourcePath(raw, kind)
}

func parseResourcePath(raw string, kind ResourceKind) (ResourcePath, bool) {
	if raw == "" {
		return ResourcePath{}, false
	}

	parts := splitTopLevel(raw, '.')
	if len(parts) == 0 {
		return ResourcePath{}, false
	}

	segments := make([]ResourcePathSegment, 0, len(parts))
	for _, part := range parts {
		segment, ok := parseResourcePathSegment(part)
		if !ok {
			return ResourcePath{}, false
		}
		segments = append(segments, segment)
	}

	if kind == ResourceKindFunc && segments[len(segments)-1].Call == nil {
		return ResourcePath{}, false
	}

	return ResourcePath{Raw: raw, Segments: segments}, true
}

func parseResourcePathSegment(raw string) (ResourcePathSegment, bool) {
	if raw == "" {
		return ResourcePathSegment{}, false
	}

	nameEnd := len(raw)
	for index, r := range raw {
		if r == '[' || r == '(' {
			nameEnd = index
			break
		}
	}

	name := raw[:nameEnd]
	if name == "" {
		return ResourcePathSegment{}, false
	}

	segment := ResourcePathSegment{Name: name}
	rest := raw[nameEnd:]
	for rest != "" {
		switch {
		case strings.HasPrefix(rest, "["):
			content, remaining, ok := takeBalanced(rest, '[', ']')
			if !ok {
				return ResourcePathSegment{}, false
			}

			scalar, ok := parseResourceScalar(content)
			if !ok {
				return ResourcePathSegment{}, false
			}
			segment.Scalars = append(segment.Scalars, scalar)
			rest = remaining
		case strings.HasPrefix(rest, "("):
			content, remaining, ok := takeBalanced(rest, '(', ')')
			if !ok || remaining != "" {
				return ResourcePathSegment{}, false
			}

			args, ok := parseResourceCallArgs(content)
			if !ok {
				return ResourcePathSegment{}, false
			}
			segment.Call = &ResourceCall{Name: name, Args: args}
			rest = remaining
		default:
			return ResourcePathSegment{}, false
		}
	}

	return segment, true
}

func parseResourceScalar(raw string) (ResourceScalar, bool) {
	if raw == "" {
		return ResourceScalar{}, false
	}

	// A raw scalar starting with ( is a lambda expression. Lambdas are resolved
	// by the substrate via string substitution before parseResourcePath is called,
	// so the parser should never receive a scalar that starts with (.
	if strings.HasPrefix(raw, "(") {
		return ResourceScalar{}, false
	}

	if key, value, ok := cutTopLevel(raw, ':'); ok {
		if key == "" || value == "" {
			return ResourceScalar{}, false
		}

		return ResourceScalar{Kind: ResourceScalarKeyValue, Key: key, Value: value}, true
	}

	if call, ok := parseResourceCall(raw); ok {
		return ResourceScalar{Kind: ResourceScalarCall, Call: &call}, true
	}

	if _, err := strconv.ParseFloat(raw, 64); err == nil {
		return ResourceScalar{Kind: ResourceScalarNumber, Value: raw}, true
	}

	return ResourceScalar{Kind: ResourceScalarKey, Key: raw}, true
}

func parseResourceCall(raw string) (ResourceCall, bool) {
	open := strings.IndexRune(raw, '(')
	if open <= 0 {
		return ResourceCall{}, false
	}

	content, remaining, ok := takeBalanced(raw[open:], '(', ')')
	if !ok || remaining != "" {
		return ResourceCall{}, false
	}

	args, ok := parseResourceCallArgs(content)
	if !ok {
		return ResourceCall{}, false
	}

	return ResourceCall{Name: raw[:open], Args: args}, true
}

func parseResourceCallArgs(raw string) ([]ResourceScalar, bool) {
	if raw == "" {
		return nil, true
	}

	parts := splitTopLevel(raw, ',')
	args := make([]ResourceScalar, 0, len(parts))
	for _, part := range parts {
		arg, ok := parseResourceScalar(part)
		if !ok {
			return nil, false
		}
		args = append(args, arg)
	}

	return args, true
}

func splitTopLevel(raw string, separator rune) []string {
	var parts []string
	start := 0
	brackets := 0
	parens := 0
	for index, r := range raw {
		switch r {
		case '[':
			brackets++
		case ']':
			brackets--
		case '(':
			parens++
		case ')':
			parens--
		default:
			if r == separator && brackets == 0 && parens == 0 {
				parts = append(parts, raw[start:index])
				start = index + len(string(r))
			}
		}
	}
	parts = append(parts, raw[start:])

	return parts
}

func cutTopLevel(raw string, separator rune) (string, string, bool) {
	brackets := 0
	parens := 0
	for index, r := range raw {
		switch r {
		case '[':
			brackets++
		case ']':
			brackets--
		case '(':
			parens++
		case ')':
			parens--
		default:
			if r == separator && brackets == 0 && parens == 0 {
				return raw[:index], raw[index+len(string(r)):], true
			}
		}
	}

	return "", "", false
}

// pathContainsLambda reports whether s contains at least one lambda expression:
// a '(' that is NOT immediately preceded by an identifier character (letter,
// digit, '-', or '_'). Such a '(' starts a lambda; one that follows an
// identifier is a function-call parenthesis.
func pathContainsLambda(s string) bool {
	for i, ch := range s {
		if ch != '(' {
			continue
		}
		if i == 0 {
			return true
		}
		prev := s[i-1]
		if !isIdentChar(prev) {
			return true
		}
	}
	return false
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

func takeBalanced(raw string, open, close rune) (string, string, bool) {
	if raw == "" || []rune(raw)[0] != open {
		return "", "", false
	}

	depth := 0
	contentStart := len(string(open))
	for index, r := range raw {
		switch r {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return raw[contentStart:index], raw[index+len(string(r)):], true
			}
		}
	}

	return "", "", false
}

func additionalPropsString(props map[string]string) string {
	if len(props) == 0 {
		return ""
	}

	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+props[key])
	}

	return strings.Join(values, "&")
}

func (pattern Hypha) Satisfies(hypha Hypha) bool {
	if !pattern.URL || !hypha.URL {
		return false
	}
	if !matchHyphaPart(pattern.Type, hypha.Type) {
		return false
	}
	if !matchHyphaPart(pattern.PackageID, hypha.PackageID) {
		return false
	}
	if !matchHyphaPart(pattern.ModuleID, hypha.ModuleID) {
		return false
	}
	if pattern.ResourceKind != "" && pattern.ResourceKind != hypha.ResourceKind {
		return false
	}
	if !matchHyphaPart(pattern.ResourcePath.String(), hypha.ResourcePath.String()) {
		return false
	}
	for key, patVal := range pattern.AdditionalProps {
		if !matchHyphaPart(patVal, hypha.AdditionalProps[key]) {
			return false
		}
	}

	return true
}

func matchHyphaPart(pattern, value string) bool {
	if pattern == "" || pattern == "$" {
		return true
	}
	if after, ok := strings.CutPrefix(pattern, "$"); ok {
		return strings.HasSuffix(value, after)
	}
	if before, ok := strings.CutSuffix(pattern, "$"); ok {
		return strings.HasPrefix(value, before)
	}

	return pattern == value
}

func indexAny(value, chars string) int {
	for index, r := range value {
		if strings.ContainsRune(chars, r) {
			return index
		}
	}

	return -1
}
