package soil

import (
	"fmt"
	"testing"
)

func TestHyphaDetectsMushroomURLAfterWhitespaceNormalization(t *testing.T) {
	s := Soil{}

	hypha := s.Hypha(" \t\n*\u200bpkg: json / github.com/ahmetson/hello-world # main ? *func = greeting() & lang = en ")

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
	if hypha.ResourceValue != "greeting()" {
		t.Fatalf("ResourceValue = %q, want %q", hypha.ResourceValue, "greeting()")
	}
	if hypha.AdditionalProps["lang"] != "en" {
		t.Fatalf("AdditionalProps[lang] = %q, want %q", hypha.AdditionalProps["lang"], "en")
	}
}

func TestHyphaDetectsNonURL(t *testing.T) {
	hypha := (&Soil{}).Hypha("hello world")

	if hypha.URL {
		t.Fatal("URL = true, want false")
	}
	if hypha.Path != "hello world" {
		t.Fatalf("Path = %q, want %q", hypha.Path, "hello world")
	}
}

func TestHyphaParsesModuleLazyLoad(t *testing.T) {
	hypha := (&Soil{}).Hypha("pkg:golang/github.com/ahmetson/hello-world#*main?func=greeting()")

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
	hypha := (&Soil{}).Hypha("hello world")

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
		ResourceValue:   "greeting()",
		AdditionalProps: map[string]string{
			"lang": "en",
			"case": "1",
		},
	}

	const want = "pkg:golang/github.com/ahmetson/hello-world#main?*func=greeting()&case=1&lang=en"
	if got := hypha.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestHyphaStringUsesAnyMarkersForMissingPackageAndModule(t *testing.T) {
	hypha := Hypha{
		URL:           true,
		ResourceKind:  ResourceKindVar,
		ResourceValue: "services[0]",
	}

	const want = "pkg:$#$?var=services[0]"
	if got := hypha.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestHyphaStringUsesTypeWithAnyPackage(t *testing.T) {
	hypha := Hypha{
		URL:           true,
		Type:          "json",
		ResourceKind:  ResourceKindVar,
		ResourceValue: "port",
	}

	const want = "pkg:json$#$?var=port"
	if got := hypha.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
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
		resource := resources[i%len(resources)]
		module := modules[i%len(modules)]
		typ := types[i%len(types)]
		packageID := packages[i%len(packages)]

		input := fmt.Sprintf("pkg:%s/%s#%s?%s%s=%s&case=%d",
			typ,
			packageID,
			module,
			derefMarker(resourceDereference),
			resource.kind,
			resource.value,
			i,
		)
		if packageID == "$" {
			input = fmt.Sprintf("pkg:%s$#%s?%s%s=%s&case=%d",
				typ,
				module,
				derefMarker(resourceDereference),
				resource.kind,
				resource.value,
				i,
			)
		}
		if prefixDereference {
			input = "*" + input
		}
		if i%2 == 0 {
			input = addWhitespaceNoise(input)
		}

		hypha := s.Hypha(input)
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
		if hypha.ResourceValue != resource.value {
			t.Fatalf("case %d: ResourceValue = %q, want %q", i, hypha.ResourceValue, resource.value)
		}
		if hypha.AdditionalProps["case"] != fmt.Sprintf("%d", i) {
			t.Fatalf("case %d: AdditionalProps[case] = %q, want %q", i, hypha.AdditionalProps["case"], fmt.Sprintf("%d", i))
		}

		wantDereference := prefixDereference || resourceDereference || module[0] == '*'
		if hypha.Dereference != wantDereference {
			t.Fatalf("case %d: Dereference = %v, want %v", i, hypha.Dereference, wantDereference)
		}

		wantDereferenceType := DereferenceType("")
		switch {
		case module[0] == '*':
			wantDereferenceType = DereferenceTypeModule
		case prefixDereference || resourceDereference:
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
