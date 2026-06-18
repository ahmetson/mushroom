package mushroom

import (
	"fmt"
	"testing"
)

func TestHyphaDetectsMushroomURLAfterWhitespaceNormalization(t *testing.T) {
	s := Soil{}

	hypha, err := s.Hypha(" \t\n*\u200bpkg: json / github.com/ahmetson/hello-world # main ? func = greeting() & lang = en ")
	if err != nil {
		t.Fatalf("Hypha returned error: %v", err)
	}

	if !hypha.URL {
		t.Fatal("URL = false, want true")
	}
	if !hypha.Dereference {
		t.Fatal("Dereference = false, want true")
	}
	if hypha.DereferenceType != DereferenceTypeResource {
		t.Fatalf("DereferenceType = %q, want %q", hypha.DereferenceType, DereferenceTypeResource)
	}
	if hypha.Type != "json" {
		t.Fatalf("Type = %q, want %q", hypha.Type, "json")
	}
	if hypha.PackageID != "github.com/ahmetson/hello-world" {
		t.Fatalf("PackageID = %q, want %q", hypha.PackageID, "github.com/ahmetson/hello-world")
	}
	if hypha.ModuleID != "main" {
		t.Fatalf("ModuleID = %q, want %q", hypha.ModuleID, "main")
	}
	if hypha.ResourceKind != ResourceKindFunc {
		t.Fatalf("ResourceKind = %q, want %q", hypha.ResourceKind, ResourceKindFunc)
	}
	if hypha.ResourcePath.String() != "greeting()" {
		t.Fatalf("ResourcePath = %q, want %q", hypha.ResourcePath, "greeting()")
	}
	if len(hypha.ResourcePath.Segments) != 1 || hypha.ResourcePath.Segments[0].Call == nil {
		t.Fatal("ResourcePath did not store greeting() as a call segment")
	}
	if hypha.AdditionalProps["lang"] != "en" {
		t.Fatalf("AdditionalProps[lang] = %q, want %q", hypha.AdditionalProps["lang"], "en")
	}
}

func TestHyphaDetectsNonURL(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("hello world")

	if hypha.URL {
		t.Fatal("URL = true, want false")
	}
	if hypha.Path != "hello world" {
		t.Fatalf("Path = %q, want %q", hypha.Path, "hello world")
	}
}

func TestHyphaParsesModuleLazyLoad(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("pkg:golang/github.com/ahmetson/hello-world#*main?func=greeting()")

	if !hypha.Dereference {
		t.Fatal("Dereference = false, want true")
	}
	if hypha.DereferenceType != DereferenceTypeModule {
		t.Fatalf("DereferenceType = %q, want %q", hypha.DereferenceType, DereferenceTypeModule)
	}
	if hypha.ModuleID != "main" {
		t.Fatalf("ModuleID = %q, want %q", hypha.ModuleID, "main")
	}
}

func TestHyphaStringReturnsPathForNonURL(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("hello world")

	if got := hypha.String(); got != "hello world" {
		t.Fatalf("String() = %q, want %q", got, "hello world")
	}
}

func TestHyphaStringReturnsFullPath(t *testing.T) {
	hypha := Hypha{
		URL:             true,
		Type:            "golang",
		PackageID:       "github.com/ahmetson/hello-world",
		ModuleID:        "main",
		Dereference:     true,
		DereferenceType: DereferenceTypeResource,
		ResourceKind:    ResourceKindFunc,
		ResourcePath:    mustResourcePath(t, "greeting()", ResourceKindFunc),
		AdditionalProps: map[string]string{
			"lang": "en",
			"case": "1",
		},
	}

	const want = "*pkg:golang/github.com/ahmetson/hello-world#main?func=greeting()&case=1&lang=en"
	if got := hypha.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestHyphaStringUsesAnyMarkersForMissingPackageAndModule(t *testing.T) {
	hypha := Hypha{
		URL:          true,
		ResourceKind: ResourceKindVar,
		ResourcePath: mustResourcePath(t, "services[0]", ResourceKindVar),
	}

	const want = "pkg:$#$?var=services[0]"
	if got := hypha.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestHyphaStringUsesTypeWithAnyPackage(t *testing.T) {
	hypha := Hypha{
		URL:          true,
		Type:         "json",
		ResourceKind: ResourceKindVar,
		ResourcePath: mustResourcePath(t, "port", ResourceKindVar),
	}

	const want = "pkg:json$#$?var=port"
	if got := hypha.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestHyphaParsesResourcePathScalars(t *testing.T) {
	hypha, err := (&Soil{}).Hypha("pkg:$?var=path0[key:$].path1[key1:value1]")
	if err != nil {
		t.Fatalf("Hypha returned error: %v", err)
	}

	if hypha.ResourcePath.String() != "path0[key:$].path1[key1:value1]" {
		t.Fatalf("ResourcePath = %q", hypha.ResourcePath.String())
	}
	if len(hypha.ResourcePath.Segments) != 2 {
		t.Fatalf("len(Segments) = %d, want %d", len(hypha.ResourcePath.Segments), 2)
	}

	first := hypha.ResourcePath.Segments[0]
	if first.Name != "path0" {
		t.Fatalf("first.Name = %q, want %q", first.Name, "path0")
	}
	if len(first.Scalars) != 1 || first.Scalars[0].Kind != ResourceScalarKeyValue {
		t.Fatalf("first.Scalars = %#v, want key-value scalar", first.Scalars)
	}
	if first.Scalars[0].Key != "key" || first.Scalars[0].Value != "$" {
		t.Fatalf("first scalar = %#v, want key:$", first.Scalars[0])
	}

	second := hypha.ResourcePath.Segments[1]
	if len(second.Scalars) != 1 || second.Scalars[0].Kind != ResourceScalarKeyValue {
		t.Fatalf("second.Scalars = %#v, want key-value scalar", second.Scalars)
	}
	if second.Scalars[0].Key != "key1" || second.Scalars[0].Value != "value1" {
		t.Fatalf("second scalar = %#v, want key1:value1", second.Scalars[0])
	}

	// A dereference lambda with no colony in the soil must return an error.
	if _, err := (&Soil{}).Hypha("pkg:$?var=path0[key:$].path1[key1:(*pkg:$?var=name)]"); err == nil {
		t.Fatal("Hypha with unresolvable dereference lambda: want error, got nil")
	}
}

func TestHyphaParsesCallResourcePath(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("pkg:$?func=hello.world.first()")

	last := hypha.ResourcePath.Segments[len(hypha.ResourcePath.Segments)-1]
	if last.Call == nil {
		t.Fatal("last segment Call = nil, want call")
	}
	if last.Call.Name != "first" {
		t.Fatalf("Call.Name = %q, want %q", last.Call.Name, "first")
	}
}

func TestHyphaRejectsFuncWithoutCall(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("pkg:$?func=fooBar")

	if hypha.ResourceKind != "" {
		t.Fatalf("ResourceKind = %q, want empty for func without call", hypha.ResourceKind)
	}
}

func TestHyphaAcceptsFuncCall(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("pkg:$?func=fooBar()")

	if hypha.ResourceKind != ResourceKindFunc {
		t.Fatalf("ResourceKind = %q, want %q", hypha.ResourceKind, ResourceKindFunc)
	}
	if hypha.ResourcePath.String() != "fooBar()" {
		t.Fatalf("ResourcePath = %q, want %q", hypha.ResourcePath.String(), "fooBar()")
	}
	if len(hypha.ResourcePath.Segments) != 1 || hypha.ResourcePath.Segments[0].Call == nil {
		t.Fatal("ResourcePath did not store fooBar() as a call")
	}
}

func TestHyphaParsesFuncCallArguments(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("pkg:$?func=fooBar(arg:arg-value,name:$)")

	call := hypha.ResourcePath.Segments[0].Call
	if call == nil {
		t.Fatal("Call = nil, want parsed call")
	}
	if call.Name != "fooBar" {
		t.Fatalf("Call.Name = %q, want %q", call.Name, "fooBar")
	}
	if len(call.Args) != 2 {
		t.Fatalf("len(Call.Args) = %d, want %d", len(call.Args), 2)
	}
	if call.Args[0].Kind != ResourceScalarKeyValue || call.Args[0].Key != "arg" || call.Args[0].Value != "arg-value" {
		t.Fatalf("Call.Args[0] = %#v, want arg:arg-value", call.Args[0])
	}
	if call.Args[1].Kind != ResourceScalarKeyValue || call.Args[1].Key != "name" || call.Args[1].Value != "$" {
		t.Fatalf("Call.Args[1] = %#v, want name:$", call.Args[1])
	}
}

func TestHyphaRejectsFuncResourceWithoutFinalCall(t *testing.T) {
	hypha, _ := (&Soil{}).Hypha("pkg:$?func=hello.world")

	if hypha.ResourceKind != "" {
		t.Fatalf("ResourceKind = %q, want empty for invalid func path", hypha.ResourceKind)
	}
}

func TestHyphaFillsFromDefault(t *testing.T) {
	soil := &Soil{}
	def, _ := soil.Hypha("pkg:json$#config.json")

	hypha, _ := soil.Hypha("pkg:$?var=services", def)

	if hypha.Type != "json" {
		t.Fatalf("Type = %q, want %q", hypha.Type, "json")
	}
	if hypha.ModuleID != "config.json" {
		t.Fatalf("ModuleID = %q, want %q", hypha.ModuleID, "config.json")
	}
	if hypha.ResourceKind != ResourceKindVar {
		t.Fatalf("ResourceKind = %q, want %q", hypha.ResourceKind, ResourceKindVar)
	}
	const want = "pkg:json$#config.json?var=services"
	if hypha.String() != want {
		t.Fatalf("String() = %q, want %q", hypha.String(), want)
	}
}

func TestHyphaDefaultDoesNotOverrideConcreteParts(t *testing.T) {
	soil := &Soil{}
	def, _ := soil.Hypha("pkg:json$#config.json")

	hypha, _ := soil.Hypha("pkg:golang/github.com/example/app#main.go?var=port", def)

	if hypha.Type != "golang" {
		t.Fatalf("Type = %q, want %q", hypha.Type, "golang")
	}
	if hypha.PackageID != "github.com/example/app" {
		t.Fatalf("PackageID = %q, want %q", hypha.PackageID, "github.com/example/app")
	}
	if hypha.ModuleID != "main.go" {
		t.Fatalf("ModuleID = %q, want %q", hypha.ModuleID, "main.go")
	}
}

func TestHyphaDefaultWithFuncCallIsRejected(t *testing.T) {
	soil := &Soil{}
	// A default with a function call in its resource path should not be applied.
	def, _ := soil.Hypha("pkg:json$#config.json?func=validate()")

	hypha, _ := soil.Hypha("pkg:$?var=port", def)

	// Type and ModuleID should remain empty/wildcard because the default was invalid.
	if hypha.Type != "" {
		t.Fatalf("Type = %q, want empty (invalid default should be ignored)", hypha.Type)
	}
}

func TestHyphaSatisfiesPattern(t *testing.T) {
	soil := &Soil{}
	pattern, _ := soil.Hypha("pkg:json/$#$.json")
	value, _ := soil.Hypha("pkg:json/github.com/example/config#app.json?var=port")

	if !pattern.Satisfies(value) {
		t.Fatalf("%q should satisfy %q", value.String(), pattern.String())
	}

	nonJSON, _ := soil.Hypha("pkg:json/github.com/example/config#app.txt?var=port")
	if pattern.Satisfies(nonJSON) {
		t.Fatalf("%q should not satisfy %q", nonJSON.String(), pattern.String())
	}
}

func TestSoilRecognizesSubstratePattern(t *testing.T) {
	soil := &Soil{}
	substrate := fakeSubstrate{url: "pkg:json/$#$.json"}
	if err := soil.AddSubstrate(substrate); err != nil {
		t.Fatalf("AddSubstrate returned error: %v", err)
	}

	rh, _ := soil.Hypha("pkg:json/github.com/example/config#app.json?var=port")
	_, got, err := soil.Recognize(rh)
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}
	if got.MushroomURL() != substrate.MushroomURL() {
		t.Fatalf("Recognize returned %q, want %q", got.MushroomURL(), substrate.MushroomURL())
	}
}

func TestSoilRejectsDereferenceSubstratePattern(t *testing.T) {
	soil := &Soil{}
	substrate := fakeSubstrate{url: "*pkg:json/$#$.json"}

	if err := soil.AddSubstrate(substrate); err == nil {
		t.Fatal("AddSubstrate returned nil error, want dereference error")
	}
}

func TestHyphaParsesOneHundredURLVariations(t *testing.T) {
	s := Soil{}
	types := []string{"golang", "json", "npm", "pypi"}
	packages := []string{
		"github.com/ahmetson/hello-world",
		"$",
		"@ahmetson/hello-world",
		"hello-world",
		"/absolute-file-path/file-name.json",
	}
	modules := []string{"main", "*main", "src/index.js", "hello_world/main.py"}
	resources := []struct {
		kind  ResourceKind
		value string
	}{
		{ResourceKindVar, "services[0]"},
		{ResourceKindFunc, "greeting()"},
		{ResourceKindObj, "Endpoint"},
		{ResourceKindVar, "port"},
	}

	for i := 0; i < 100; i++ {
		prefixDereference := i%10 == 0
		resourceDereference := i%3 == 0
		// All dereferences are expressed via *pkg: prefix; ?*kind= is not supported.
		addDerefPrefix := prefixDereference || resourceDereference
		resource := resources[i%len(resources)]
		module := modules[i%len(modules)]
		typ := types[i%len(types)]
		packageID := packages[i%len(packages)]

		input := fmt.Sprintf("pkg:%s/%s#%s?%s=%s&case=%d",
			typ,
			packageID,
			module,
			resource.kind,
			resource.value,
			i,
		)
		if packageID == "$" {
			input = fmt.Sprintf("pkg:%s$#%s?%s=%s&case=%d",
				typ,
				module,
				resource.kind,
				resource.value,
				i,
			)
		}
		if addDerefPrefix {
			input = "*" + input
		}
		if i%2 == 0 {
			input = addWhitespaceNoise(input)
		}

		hypha, _ := s.Hypha(input)
		if !hypha.URL {
			t.Fatalf("case %d: URL = false, want true for %q", i, input)
		}
		if hypha.Type != typ {
			t.Fatalf("case %d: Type = %q, want %q", i, hypha.Type, typ)
		}
		if hypha.PackageID != packageID {
			t.Fatalf("case %d: PackageID = %q, want %q", i, hypha.PackageID, packageID)
		}
		if hypha.ModuleID != trimModuleDereference(module) {
			t.Fatalf("case %d: ModuleID = %q, want %q", i, hypha.ModuleID, trimModuleDereference(module))
		}
		if hypha.ResourceKind != resource.kind {
			t.Fatalf("case %d: ResourceKind = %q, want %q", i, hypha.ResourceKind, resource.kind)
		}
		if hypha.ResourcePath.String() != resource.value {
			t.Fatalf("case %d: ResourcePath = %q, want %q", i, hypha.ResourcePath, resource.value)
		}
		if hypha.AdditionalProps["case"] != fmt.Sprintf("%d", i) {
			t.Fatalf("case %d: AdditionalProps[case] = %q, want %q", i, hypha.AdditionalProps["case"], fmt.Sprintf("%d", i))
		}

		wantDereference := addDerefPrefix || module[0] == '*'
		if hypha.Dereference != wantDereference {
			t.Fatalf("case %d: Dereference = %v, want %v", i, hypha.Dereference, wantDereference)
		}

		wantDereferenceType := DereferenceType("")
		switch {
		case module[0] == '*':
			wantDereferenceType = DereferenceTypeModule
		case addDerefPrefix:
			wantDereferenceType = DereferenceTypeResource
		}
		if hypha.DereferenceType != wantDereferenceType {
			t.Fatalf("case %d: DereferenceType = %q, want %q", i, hypha.DereferenceType, wantDereferenceType)
		}
	}
}

func derefMarker(dereference bool) string {
	if dereference {
		return "*"
	}

	return ""
}

func trimModuleDereference(module string) string {
	if module[0] == '*' {
		return module[1:]
	}

	return module
}

func addWhitespaceNoise(input string) string {
	return " \t\n" + input[:4] + "\u200b " + input[4:8] + "\u2060" + input[8:] + " \n"
}

func mustResourcePath(t *testing.T, raw string, kind ResourceKind) ResourcePath {
	t.Helper()

	path, ok := parseResourcePath(raw, kind)
	if !ok {
		t.Fatalf("parseResourcePath(%q, %q) failed", raw, kind)
	}

	return path
}

type fakeSubstrate struct {
	url string
}

func (substrate fakeSubstrate) Digest(Hypha, any, *Soil) (Mycelium, error) {
	return nil, nil
}

func (substrate fakeSubstrate) MushroomURL() string {
	return substrate.url
}

func (substrate fakeSubstrate) Forage(Hypha) (any, error) {
	return nil, nil
}

func (substrate fakeSubstrate) Sow(Hypha, any) error {
	return nil
}

func TestParentResourceURLSymbolic(t *testing.T) {
	parent, ok := ParentResourceURL("main")
	if ok {
		t.Fatal("ok = true, want false for symbolic url")
	}
	if parent != "main" {
		t.Fatalf("parent = %q, want %q", parent, "main")
	}
}

func TestParentResourceURLNoResourcePath(t *testing.T) {
	const url = "pkg:golang/github.com/ahmetson/hello-world#main"
	parent, ok := ParentResourceURL(url)
	if ok {
		t.Fatal("ok = true, want false")
	}
	if parent != url {
		t.Fatalf("parent = %q, want %q", parent, url)
	}
}

func TestParentResourceURLSinglePlainSegment(t *testing.T) {
	const url = "pkg:$?var=services"
	parent, ok := ParentResourceURL(url)
	if ok {
		t.Fatal("ok = true, want false")
	}
	const want = "pkg:$#$?var=services"
	if parent != want {
		t.Fatalf("parent = %q, want %q", parent, want)
	}
}

func TestParentResourceURLSingleIndexedSegment(t *testing.T) {
	const (
		url  = "pkg:$?var=services[name:proxy]"
		want = "pkg:$?var=services"
	)
	parent, ok := ParentResourceURL(url)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if parent != want {
		t.Fatalf("parent = %q, want %q", parent, want)
	}
}

func TestParentResourceURLMultiSegment(t *testing.T) {
	const (
		url  = "*pkg:$?var=services[name:proxy].handlers[category:main].outbounds"
		want = "*pkg:$?var=services[name:proxy].handlers[category:main]"
	)
	parent, ok := ParentResourceURL(url)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if parent != want {
		t.Fatalf("parent = %q, want %q", parent, want)
	}
}

func TestParentResourceURLTrimsHandlerSegment(t *testing.T) {
	const (
		url  = "*pkg:$?var=services[name:proxy].handlers[category:main]"
		want = "*pkg:$?var=services[name:proxy].handlers"
	)
	parent, ok := ParentResourceURL(url)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if parent != want {
		t.Fatalf("parent = %q, want %q", parent, want)
	}
}

func TestHyphaParentResourceURL(t *testing.T) {
	hypha, err := (&Soil{}).Hypha("*pkg:$?var=services[name:proxy].handlers[category:main].outbounds")
	if err != nil {
		t.Fatalf("Hypha returned error: %v", err)
	}

	parent, ok := hypha.ParentResourceURL()
	if !ok {
		t.Fatal("ok = false, want true")
	}
	const want = "*pkg:$?var=services[name:proxy].handlers[category:main]"
	if parent != want {
		t.Fatalf("parent = %q, want %q", parent, want)
	}
}
