package soil

import (
	"sort"
	"strings"
	"unicode"
)

type Soil struct{}

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

type Hypha struct {
	Path            string // The original path: "some random data" | "pkg:$?var=services[0]"
	Type            string // Package type same as package-url
	Dereference     bool   // Whether the path is a dereference: true | false
	DereferenceType DereferenceType
	PackageID       string // Package url with its namespace and name
	ModuleID        string // Module name or file name
	ResourceKind    ResourceKind
	ResourceValue   string
	AdditionalProps map[string]string
	URL             bool // if path has `pkg:` prefixed its a mushroom url. Otherwise its just a symbol
}

func (*Soil) Hypha(path string) Hypha {
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

	return hypha
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
		builder.WriteString(hypha.ResourceValue)
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
		hypha.ResourceKind = ResourceKind(kind)
		hypha.ResourceValue = value
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

func indexAny(value, chars string) int {
	for index, r := range value {
		if strings.ContainsRune(chars, r) {
			return index
		}
	}

	return -1
}
