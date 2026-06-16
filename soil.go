package mushroom

import (
	"errors"
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

func (soil *Soil) AddSubstrate(substrate Substrate) error {
	if substrate == nil {
		return ErrInvalidSubstrate
	}

	hypha := soil.Hypha(substrate.MushroomURL())
	if !hypha.URL || hypha.Dereference {
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

func (soil *Soil) Substrates() []Substrate {
	return append([]Substrate(nil), soil.substrates...)
}

func (soil *Soil) Colony() []Mycelium {
	return append([]Mycelium(nil), soil.colonies...)
}

func (soil *Soil) Recognize(path string) (Mycelium, Substrate, error) {
	hypha := soil.Hypha(path)
	for _, mycelium := range soil.colonies {
		if soil.Hypha(mycelium.MushroomURL()).Satisfies(hypha) {
			return mycelium, nil, nil
		}
	}

	for _, substrate := range soil.substrates {
		if soil.Hypha(substrate.MushroomURL()).Satisfies(hypha) {
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
	// RawResourcePath is the unresolved resource path string exactly as it
	// appears after the kind= separator, before any lambda substitution. It is
	// non-empty whenever ResourceKind is set. Substrates use this to run lambda
	// resolution before re-parsing the concrete path for lookup.
	RawResourcePath string
	AdditionalProps map[string]string
	URL             bool // if path has `pkg:` prefixed its a mushroom url. Otherwise its just a symbol
}

// Hypha parses path into a structured Hypha.
// An optional defaults Hypha can be provided; any empty or wildcard ($) field
// in the parsed result is filled from defaults, provided defaults contains no
// lambdas and no function-call segments in its resource path.
func (*Soil) Hypha(path string, defaults ...Hypha) Hypha {
	hypha := Hypha{Path: path}
	normalized := stripIgnoredRunes(path)

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
		return hypha
	}

	packageAndSections := parseTypeAndPackage(normalized, &hypha)
	parseSections(packageAndSections, &hypha)

	if len(defaults) > 0 {
		fillHyphaFromDefault(&hypha, defaults[0])
	}

	return hypha
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
		if hypha.Dereference && hypha.DereferenceType == DereferenceTypeResource {
			builder.WriteString("*")
		}
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

	if strings.HasPrefix(resourcePart, "*") {
		hypha.Dereference = true
		if hypha.DereferenceType != DereferenceTypeModule {
			hypha.DereferenceType = DereferenceTypeResource
		}
		resourcePart = strings.TrimPrefix(resourcePart, "*")
	}

	kind, value, ok := strings.Cut(resourcePart, "=")
	if !ok {
		return
	}

	switch kind {
	case string(ResourceKindVar), string(ResourceKindFunc), string(ResourceKindObj):
		resourceKind := ResourceKind(kind)
		// Paths that contain lambda expressions (…) are deferred: we store the
		// raw string and skip immediate structural validation because lambdas are
		// resolved at runtime by the substrate before the path is parsed.
		// A ( is a lambda start only when it is NOT preceded by an identifier
		// character (letter, digit, _ or -). Preceded by one of those it is a
		// function-call parenthesis and does not indicate a lambda.
		if pathContainsLambda(value) {
			hypha.ResourceKind = resourceKind
			hypha.RawResourcePath = value
			return
		}
		resourcePath, ok := parseResourcePath(value, resourceKind)
		if !ok {
			return
		}
		hypha.ResourceKind = resourceKind
		hypha.RawResourcePath = value
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
