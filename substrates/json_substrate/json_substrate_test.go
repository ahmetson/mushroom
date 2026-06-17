package json_substrate

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ahmetson/mushroom"
)

func TestSubstrateImplementsMushroomSubstrate(t *testing.T) {
	var _ mushroom.Substrate = (*Substrate)(nil)
}

func TestMyceliumImplementsMushroomMycelium(t *testing.T) {
	var _ mushroom.Mycelium = (*Mycelium)(nil)
}

func TestDigestAcceptsString(t *testing.T) {
	mycelium, err := Digest("pkg:json$#my-app-config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	if mycelium.Soil() == nil {
		t.Fatal("Soil() = nil, want soil")
	}
	if mycelium.Substrate() == nil {
		t.Fatal("Substrate() = nil, want substrate")
	}
	if mycelium.url.Type != "json" {
		t.Fatalf("url.Type = %q, want %q", mycelium.url.Type, "json")
	}
	if mycelium.url.ModuleID != "my-app-config.json" {
		t.Fatalf("url.ModuleID = %q, want %q", mycelium.url.ModuleID, "my-app-config.json")
	}
	if mycelium.MushroomURL() != "pkg:json$#my-app-config.json" {
		t.Fatalf("MushroomURL() = %q, want %q", mycelium.MushroomURL(), "pkg:json$#my-app-config.json")
	}
}

func TestSubstrateDigestRejectsBytes(t *testing.T) {
	soil := &mushroom.Soil{}
	substrate := &Substrate{url: soil.Hypha("pkg:json/$#$.json")}

	if _, err := substrate.Digest("pkg:json$#my-app-config.json", []byte(`{"port":8080}`), soil); err == nil {
		t.Fatal("Digest returned nil error, want unsupported bytes error")
	}
}

func TestDigestUsesProvidedSoil(t *testing.T) {
	soil := &mushroom.Soil{}
	substrate := &Substrate{url: soil.Hypha("pkg:json/$#$.json")}

	got, err := substrate.Digest("pkg:json$#my-app-config.json", `{"port":8080}`, soil)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	mycelium := got.(*Mycelium)
	if mycelium.Soil() != soil {
		t.Fatal("Soil() did not return provided soil")
	}
	if len(mycelium.Soil().Substrates()) != 0 {
		t.Fatal("provided soil should not be modified with default substrates")
	}
}

func TestDigestAddsJSONSubstrateToNewSoil(t *testing.T) {
	mycelium, err := Digest("pkg:json$#my-app-config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	substrates := mycelium.Soil().Substrates()
	if len(substrates) != 1 {
		t.Fatalf("len(Substrates()) = %d, want %d", len(substrates), 1)
	}
	if substrates[0].MushroomURL() != "pkg:json$#$.json" {
		t.Fatalf("Substrates()[0].MushroomURL() = %q, want %q", substrates[0].MushroomURL(), "pkg:json$#$.json")
	}
}

func TestJSONSubstratePatternRecognizesJSONModules(t *testing.T) {
	mycelium, err := Digest("pkg:json$#my-app-config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	_, substrate, err := mycelium.Soil().Recognize("pkg:json/json_dir#my-app-config.json")
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}
	if substrate == nil {
		t.Fatal("Recognize returned nil substrate")
	}
}

func TestJSONSubstratePatternRejectsOtherJSONLikeTypes(t *testing.T) {
	mycelium, err := Digest("pkg:json$#my-app-config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	if _, _, err := mycelium.Soil().Recognize("pkg:json-ad/file_dir#config.json"); err == nil {
		t.Fatal("Recognize returned nil error, want json-ad to fail")
	}
}

func TestSporeReturnsJSONValue(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"services":[{"hostname":"localhost","port":8080}]}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("pkg:$?*var=services[0].port")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}

	number, ok := got.(json.Number)
	if !ok {
		t.Fatalf("Spore returned %T, want json.Number", got)
	}
	if number.String() != "8080" {
		t.Fatalf("Spore returned %q, want %q", number.String(), "8080")
	}
}

func TestSporeReturnsPortVar(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("pkg:$?*var=port")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}

	number, ok := got.(json.Number)
	if !ok {
		t.Fatalf("Spore returned %T, want json.Number", got)
	}
	if number.String() != "8080" {
		t.Fatalf("Spore returned %q, want %q", number.String(), "8080")
	}
}

func TestSporeReturnsSymbolic(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("auth_proxy")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}
	if got != "auth_proxy" {
		t.Fatalf("Spore returned %v, want %q", got, "auth_proxy")
	}
}

func TestSporeRejectsLinkURL(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	_, err = mycelium.Spore("pkg:$?var=port")
	if err == nil {
		t.Fatal("Spore returned nil error for link URL, want error")
	}
}

func TestSporeRejectsModuleDereference(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	_, err = mycelium.Spore("pkg:json$#*config.json")
	if err == nil {
		t.Fatal("Spore returned nil error for module dereference, want error")
	}
}

func TestSporeTraversesArrayRootNameKey(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `[{"name":"alpha","key":"match"},{"name":"beta"}]`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("pkg:$?*var=name[key]")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}
	if got != "alpha" {
		t.Fatalf("Spore returned %v, want %q", got, "alpha")
	}
}

func TestSporeTraversesNestedPath(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"outer":{"inner":{"port":3000}}}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("pkg:$?*var=outer.inner.port")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}

	number, ok := got.(json.Number)
	if !ok {
		t.Fatalf("Spore returned %T, want json.Number", got)
	}
	if number.String() != "3000" {
		t.Fatalf("Spore returned %q, want %q", number.String(), "3000")
	}
}

func TestSporeEvaluatesSegmentLevelLambda(t *testing.T) {
	// Lambda at segment level: (*pkg:$?var=fieldName) is resolved to the value
	// of fieldName ("port"), which is then used as the segment name, yielding
	// the same result as Spore("pkg:$?*var=port").
	mycelium, err := Digest("pkg:json$#config.json", `{"fieldName":"port","port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("pkg:$?*var=(*pkg:$?var=fieldName)")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}

	number, ok := got.(json.Number)
	if !ok {
		t.Fatalf("Spore returned %T, want json.Number", got)
	}
	if number.String() != "8080" {
		t.Fatalf("Spore returned %q, want %q", number.String(), "8080")
	}
}

func TestSporeEvaluatesDereferenceLambdaInKeyValueFilter(t *testing.T) {
	// Lambda in the value position of a key:value filter:
	// services[name:(*pkg:$?var=key)] resolves the lambda to the value of "key"
	// and uses the result as the filter value — equivalent to services[name:foo].
	mycelium, err := Digest("pkg:json$#config.json",
		`{"key":"foo","services":[{"name":"foo","port":9000},{"name":"bar","port":9001}]}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("pkg:$?*var=services[name:(*pkg:$?var=key)]")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}

	matches, ok := got.([]any)
	if !ok || len(matches) == 0 {
		t.Fatalf("Spore returned %T (len %d), want non-empty []any", got, len(matches))
	}
	first, ok := matches[0].(map[string]any)
	if !ok || first["name"] != "foo" {
		t.Fatalf("first match name = %v, want %q", first["name"], "foo")
	}
}

func TestFruitEvaluatesDereferenceLinks(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Fruit(map[string]any{
		"hostname": "localhost",
		"port":     "*pkg:json$#config.json?var=port",
	})
	if err != nil {
		t.Fatalf("Fruit returned error: %v", err)
	}

	endpoint, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Fruit returned %T, want map[string]any", got)
	}
	number, ok := endpoint["port"].(json.Number)
	if !ok {
		t.Fatalf("port = %T, want json.Number", endpoint["port"])
	}
	if number.String() != "8080" {
		t.Fatalf("port = %q, want %q", number.String(), "8080")
	}
}

func TestMineralizeReturnsJSONFormat(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{
		"hostname": "localhost",
		"port": 8080
	}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Mineralize()
	if err != nil {
		t.Fatalf("Mineralize returned error: %v", err)
	}

	const want = `{"hostname":"localhost","port":8080}`
	if got != want {
		t.Fatalf("Mineralize returned %q, want %q", got, want)
	}
}

func TestLinkRejectsDereference(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	if _, err := mycelium.Link("pkg:$?*var=port"); err == nil {
		t.Fatal("Link returned nil error, want dereference error")
	}
}

func TestDigestRejectsNonMushroomURL(t *testing.T) {
	if _, err := Digest("config.json", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want invalid URL error")
	}
}

func TestDigestRejectsDereferenceURL(t *testing.T) {
	if _, err := Digest("*pkg:json$#config.json", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want dereference URL error")
	}
}

func TestDigestRejectsNonJSONType(t *testing.T) {
	if _, err := Digest("pkg:golang$#config.json", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want invalid type error")
	}
}

func TestDigestRejectsNonJSONModuleID(t *testing.T) {
	if _, err := Digest("pkg:json$#config.txt", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want invalid module id error")
	}
}

func TestLinkFillsTypeAndModuleFromMyceliumURL(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Link("pkg:$?var=port")
	if err != nil {
		t.Fatalf("Link returned error: %v", err)
	}
	const want = "pkg:json$#config.json?var=port"
	if got != want {
		t.Fatalf("Link = %q, want %q", got, want)
	}
}

func TestLinkFillsModuleFromMyceliumURL(t *testing.T) {
	// pkg:json$?var=port has no module (#). Link fills it from the mycelium URL.
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Link("pkg:json$?var=port")
	if err != nil {
		t.Fatalf("Link returned error: %v", err)
	}
	const want = "pkg:json$#config.json?var=port"
	if got != want {
		t.Fatalf("Link = %q, want %q", got, want)
	}
}

func TestLinkRejectsNonExistentResource(t *testing.T) {
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	if _, err := mycelium.Link("pkg:$?var=nonExistent"); err == nil {
		t.Fatal("Link returned nil error, want not-found error for missing resource")
	}
}

func TestLinkRejectsURLWithoutWildcard(t *testing.T) {
	// pkg:?var=port has no $ wildcard anywhere in Type, PackageID, or ModuleID.
	// fillHyphaFromDefault must NOT fill empty fields in this case, so the URL
	// stays unrecognized and Link must return an error.
	mycelium, err := Digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	if _, err := mycelium.Link("pkg:?var=port"); err == nil {
		t.Fatal("Link(pkg:?var=port): expected error for URL with no $ wildcard, got nil")
	}
}

func TestNoPerfectionMycelium(t *testing.T) {
	raw, err := os.ReadFile("noPerfection.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	noPerfMycelium, err := Digest("pkg:json$#noPerfection.json", string(raw))
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}

	// Link to services and spore the full services array.
	if _, err := noPerfMycelium.Link("pkg:$?var=services"); err != nil {
		t.Fatalf("Link(services): %v", err)
	}
	got, err := noPerfMycelium.Spore("pkg:$?*var=services")
	if err != nil {
		t.Fatalf("Spore(services): %v", err)
	}
	services, ok := got.([]any)
	if !ok {
		t.Fatalf("Spore(services) returned %T, want []any", got)
	}
	if len(services) != 3 {
		t.Fatalf("len(services) = %d, want 3", len(services))
	}

	// Link to the first service via services[$.first()] and verify it is the first.
	if _, err := noPerfMycelium.Link("pkg:$?var=services[$.first()]"); err != nil {
		t.Fatalf("Link(services[$.first()]): %v", err)
	}
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[$.first()]")
	if err != nil {
		t.Fatalf("Spore(services[$.first()]): %v", err)
	}
	first, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Spore(services[$.first()]) returned %T, want map", got)
	}
	if first["name"] != "default-name-proxy" {
		t.Fatalf("first service name = %q, want %q", first["name"], "default-name-proxy")
	}

	// Link to the last service via services[$.last()] and verify it is the last.
	if _, err := noPerfMycelium.Link("pkg:$?var=services[$.last()]"); err != nil {
		t.Fatalf("Link(services[$.last()]): %v", err)
	}
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[$.last()]")
	if err != nil {
		t.Fatalf("Spore(services[$.last()]): %v", err)
	}
	last, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Spore(services[$.last()]) returned %T, want map", got)
	}
	if last["name"] != "hello-world" {
		t.Fatalf("last service name = %q, want %q", last["name"], "hello-world")
	}

	// Index 0 returns the same first element.
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[0]")
	if err != nil {
		t.Fatalf("Spore(services[0]): %v", err)
	}
	byIndex, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Spore(services[0]) returned %T, want map", got)
	}
	if byIndex["name"] != "default-name-proxy" {
		t.Fatalf("services[0] name = %q, want %q", byIndex["name"], "default-name-proxy")
	}

	// Index 3 does not exist: both Link and Spore must reject it.
	if _, err := noPerfMycelium.Link("pkg:$?var=services[3]"); err == nil {
		t.Fatal("Link(services[3]) returned nil error, want error for non-existent index")
	}
	if _, err := noPerfMycelium.Spore("pkg:$?*var=services[3]"); err == nil {
		t.Fatal("Spore(services[3]) returned nil error, want out-of-range error")
	}

	// Numeric indexes and $.first() / $.last() require an array.
	// services[0].type is a plain string, so further indexing must fail.
	if _, err := noPerfMycelium.Spore("pkg:$?*var=services[0].type[0]"); err == nil {
		t.Fatal("Spore(services[0].type[0]) returned nil error, want non-array error")
	}
	if _, err := noPerfMycelium.Spore("pkg:$?*var=services[0].type[$.first()]"); err == nil {
		t.Fatal("Spore(services[0].type[$.first()]) returned nil error, want non-array error")
	}
	if _, err := noPerfMycelium.Spore("pkg:$?*var=services[0].type[$.last()]"); err == nil {
		t.Fatal("Spore(services[0].type[$.last()]) returned nil error, want non-array error")
	}

	// Filter by type:Proxy returns both proxy services.
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[type:Proxy]")
	if err != nil {
		t.Fatalf("Spore(services[type:Proxy]): %v", err)
	}
	proxies, ok := got.([]any)
	if !ok {
		t.Fatalf("Spore(services[type:Proxy]) returned %T, want []any", got)
	}
	if len(proxies) != 2 {
		t.Fatalf("len(proxies) = %d, want 2", len(proxies))
	}
	for i, p := range proxies {
		obj, ok := p.(map[string]any)
		if !ok {
			t.Fatalf("proxies[%d] is %T, want map", i, p)
		}
		if obj["type"] != "Proxy" {
			t.Fatalf("proxies[%d].type = %q, want %q", i, obj["type"], "Proxy")
		}
	}

	// Filter by type:Proxy then [$.last()] returns the second proxy (entrypoint).
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[type:Proxy][$.last()]")
	if err != nil {
		t.Fatalf("Spore(services[type:Proxy][$.last()]): %v", err)
	}
	lastProxy, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Spore(services[type:Proxy][$.last()]) returned %T, want map", got)
	}
	if lastProxy["name"] != "entrypoint" {
		t.Fatalf("last proxy name = %q, want %q", lastProxy["name"], "entrypoint")
	}

	// The handlers field of outbounds[0] is a dereference URL that points to
	// the hello-world service's main handler via
	// services[name:hello-world].handlers[category:main].
	// Spore the outbound to confirm the field is still a raw dereference string.
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[0].handlers[0].outbounds[0]")
	if err != nil {
		t.Fatalf("Spore(outbound): %v", err)
	}
	outbound, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Spore(outbound) returned %T, want map", got)
	}
	const wantDeref = "*pkg:$?var=services[name:hello-world].handlers[category:main]"
	handlersRaw, ok := outbound["handlers"].([]any)
	if !ok || len(handlersRaw) != 1 {
		t.Fatalf("outbound handlers = %T (len %d), want []any of length 1", outbound["handlers"], len(handlersRaw))
	}
	if handlersRaw[0] != wantDeref {
		t.Fatalf("outbound handlers[0] = %q, want dereference string %q", handlersRaw[0], wantDeref)
	}

	// Fruit the outbound: Fruit evaluates the dereference string in the handlers
	// array by calling Spore on services[name:hello-world].handlers[category:main],
	// which drills into the hello-world service and filters its handlers by
	// category=main, returning the Replier/main handler with endpoint port 8000.
	fruited, err := noPerfMycelium.Fruit(outbound)
	if err != nil {
		t.Fatalf("Fruit(outbound): %v", err)
	}
	fruitedOutbound, ok := fruited.(map[string]any)
	if !ok {
		t.Fatalf("Fruit(outbound) returned %T, want map", fruited)
	}
	fruitedHandlers, ok := fruitedOutbound["handlers"].([]any)
	if !ok || len(fruitedHandlers) != 1 {
		t.Fatalf("fruited handlers = %T (len %d), want []any of length 1", fruitedOutbound["handlers"], len(fruitedHandlers))
	}
	handler, ok := fruitedHandlers[0].(map[string]any)
	if !ok {
		t.Fatalf("fruited handlers[0] = %T, want map (resolved handler)", fruitedHandlers[0])
	}
	if handler["type"] != "Replier" {
		t.Fatalf("handler type = %q, want %q", handler["type"], "Replier")
	}
	if handler["category"] != "main" {
		t.Fatalf("handler category = %q, want %q", handler["category"], "main")
	}
	endpoint, ok := handler["endpoint"].(map[string]any)
	if !ok {
		t.Fatalf("handler endpoint = %T, want map", handler["endpoint"])
	}
	port, ok := endpoint["port"].(json.Number)
	if !ok {
		t.Fatalf("endpoint port = %T, want json.Number", endpoint["port"])
	}
	if port.String() != "8000" {
		t.Fatalf("endpoint port = %q, want %q", port.String(), "8000")
	}

	// Lambda string substitution in scalar position.
	// (*pkg:$?var=services[0].name) is resolved to "default-name-proxy" and then
	// substituted as text, producing services[default-name-proxy]. That scalar is
	// a Key lookup — it looks for a field named "default-name-proxy" on each
	// service object. No service has that field, so Spore must fail.
	if _, err = noPerfMycelium.Spore("pkg:$?*var=services[(*pkg:$?var=services[0].name)]"); err == nil {
		t.Fatal("Spore(services[(*lambda)]): expected error — substituted key 'default-name-proxy' is not a field on any service")
	}

	// Non-dereference lambda in scalar position:
	// (pkg:$?var=services[0].name) resolves to the link string
	// "pkg:json$#noPerfection.json?var=services[0].name" and is substituted,
	// producing services[pkg:json$#noPerfection.json?var=services[0].name].
	// The colon makes it a KeyValue filter with Key="pkg" — no service has
	// that field, so Spore must fail.
	if _, err = noPerfMycelium.Spore("pkg:$?*var=services[(pkg:$?var=services[0].name)]"); err == nil {
		t.Fatal("Spore(services[(non-deref lambda)]): expected error — substituted link URL is not a valid filter")
	}

	// Plain key scalar [default-name-proxy] (no colon) looks for a field NAMED
	// "default-name-proxy" on each service object — it is NOT a name-value filter.
	// Since no service has a field with that literal name, Spore must fail.
	if _, err = noPerfMycelium.Spore("pkg:$?*var=services[default-name-proxy]"); err == nil {
		t.Fatal("Spore(services[default-name-proxy]): expected error — plain key looks for a field named 'default-name-proxy', not a service with that name")
	}

	// The correct way to filter by name value is the key:value form.
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[name:default-name-proxy]")
	if err != nil {
		t.Fatalf("Spore(services[name:default-name-proxy]): %v", err)
	}
	byNameFilter, ok := got.([]any)
	if !ok || len(byNameFilter) == 0 {
		t.Fatalf("Spore(services[name:default-name-proxy]) returned %T (len %d), want non-empty []any", got, len(byNameFilter))
	}
	firstByName, ok := byNameFilter[0].(map[string]any)
	if !ok || firstByName["name"] != "default-name-proxy" {
		t.Fatalf("services[name:default-name-proxy][0][\"name\"] = %v, want %q", firstByName["name"], "default-name-proxy")
	}

	// Lambda in the value position of a key:value filter.
	// services[name:(*pkg:$?var=services[0].name)] evaluates the dereference
	// lambda to "default-name-proxy" and then uses it as the value to filter by,
	// returning the same service as services[name:default-name-proxy].
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[name:(*pkg:$?var=services[0].name)]")
	if err != nil {
		t.Fatalf("Spore(services[name:(lambda)]): %v", err)
	}
	byLambdaFilter, ok := got.([]any)
	if !ok || len(byLambdaFilter) == 0 {
		t.Fatalf("Spore(services[name:(lambda)]) returned %T (len %d), want non-empty []any", got, len(byLambdaFilter))
	}
	firstByLambda, ok := byLambdaFilter[0].(map[string]any)
	if !ok || firstByLambda["name"] != "default-name-proxy" {
		t.Fatalf("services[name:(lambda)][0][\"name\"] = %v, want %q", firstByLambda["name"], "default-name-proxy")
	}

	// services[name:default-name-proxy.first()] — first() is INSIDE the bracket,
	// so "default-name-proxy.first()" is the literal string compared against the
	// name field. No service has that name, so the call must fail.
	if _, err = noPerfMycelium.Spore("pkg:$?*var=services[name:default-name-proxy.first()]"); err == nil {
		t.Fatal("Spore(services[name:default-name-proxy.first()]): expected error — value is a literal string, not a call")
	}

	// services[name:default-name-proxy].first() — first() is a SEPARATE segment
	// after the bracket. The filter returns []any with one matching service, then
	// first() picks its first element, yielding the service map directly.
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[name:default-name-proxy].first()")
	if err != nil {
		t.Fatalf("Spore(services[name:default-name-proxy].first()): %v", err)
	}
	firstOfFilter, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("services[name:default-name-proxy].first() returned %T, want map", got)
	}
	if firstOfFilter["name"] != "default-name-proxy" {
		t.Fatalf("services[name:default-name-proxy].first()[\"name\"] = %v, want %q", firstOfFilter["name"], "default-name-proxy")
	}

	// Spore the first service. The raw data is returned as-is: the dereference
	// string inside outbounds[0].handlers is NOT yet evaluated.
	got, err = noPerfMycelium.Spore("pkg:$?*var=services[$.first()]")
	if err != nil {
		t.Fatalf("Spore(first service): %v", err)
	}
	firstService, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Spore(first service) returned %T, want map", got)
	}
	if firstService["name"] != "default-name-proxy" {
		t.Fatalf("first service name = %q, want %q", firstService["name"], "default-name-proxy")
	}
	// Confirm the dereference is still a raw string (not yet resolved).
	rawOutbound := firstService["handlers"].([]any)[0].(map[string]any)["outbounds"].([]any)[0].(map[string]any)
	rawHandlers, ok := rawOutbound["handlers"].([]any)
	if !ok || len(rawHandlers) == 0 || rawHandlers[0] != "*pkg:$?var=services[name:hello-world].handlers[category:main]" {
		t.Fatalf("raw outbound handlers = %v, want array with dereference string as first element", rawOutbound["handlers"])
	}

	// Fruit the first service. Fruit recursively traverses the whole object
	// and evaluates any dereference strings it encounters.
	// When it reaches outbounds[0].handlers[0] it calls Spore on the dereference,
	// which drills into the hello-world service and returns its main handler.
	fruitedService, err := noPerfMycelium.Fruit(firstService)
	if err != nil {
		t.Fatalf("Fruit(first service): %v", err)
	}
	fruitedMap, ok := fruitedService.(map[string]any)
	if !ok {
		t.Fatalf("Fruit(first service) returned %T, want map", fruitedService)
	}
	resolvedOutbound := fruitedMap["handlers"].([]any)[0].(map[string]any)["outbounds"].([]any)[0].(map[string]any)
	resolvedHandlers, ok := resolvedOutbound["handlers"].([]any)
	if !ok || len(resolvedHandlers) == 0 {
		t.Fatalf("resolved outbound handlers = %T, want non-empty array", resolvedOutbound["handlers"])
	}
	resolvedHandler, ok := resolvedHandlers[0].(map[string]any)
	if !ok {
		t.Fatalf("resolved outbound handlers[0] = %T, want map", resolvedHandlers[0])
	}
	if resolvedHandler["type"] != "Replier" || resolvedHandler["category"] != "main" {
		t.Fatalf("resolved handler = {type:%q, category:%q}, want {Replier, main}",
			resolvedHandler["type"], resolvedHandler["category"])
	}
	resolvedPort := resolvedHandler["endpoint"].(map[string]any)["port"].(json.Number)
	if resolvedPort.String() != "8000" {
		t.Fatalf("resolved handler port = %q, want %q", resolvedPort.String(), "8000")
	}

	// Deep dereference path: traverse into the default-name-proxy service, pick
	// its main handler, take the first outbound, and pick the first element of
	// that outbound's handlers array.
	//
	// The handlers array holds "*pkg:$?var=services[name:hello-world].handlers[category:main]"
	// as its first element, so Spore returns that raw dereference string.
	// Fruit then evaluates it to the Replier/main handler from hello-world
	// (type=Replier, category=main, endpoint port=8000).
	deepPath := "*pkg:$?var=services[name:default-name-proxy].handlers[category:main].outbounds[$.first()].handlers[$.first()]"
	got, err = noPerfMycelium.Spore(deepPath)
	if err != nil {
		t.Fatalf("Spore(deep path): %v", err)
	}
	deepDeref, ok := got.(string)
	if !ok {
		t.Fatalf("Spore(deep path) returned %T, want string (raw dereference)", got)
	}
	if deepDeref != wantDeref {
		t.Fatalf("Spore(deep path) = %q, want dereference string %q", deepDeref, wantDeref)
	}

	// Fruit resolves the raw dereference to the actual handler data.
	resolvedDeep, err := noPerfMycelium.Fruit(deepDeref)
	if err != nil {
		t.Fatalf("Fruit(deep dereference): %v", err)
	}
	deepHandler, ok := resolvedDeep.(map[string]any)
	if !ok {
		t.Fatalf("Fruit(deep dereference) returned %T, want map", resolvedDeep)
	}
	if deepHandler["type"] != "Replier" || deepHandler["category"] != "main" {
		t.Fatalf("deep handler = {type:%q, category:%q}, want {Replier, main}",
			deepHandler["type"], deepHandler["category"])
	}
	deepPort := deepHandler["endpoint"].(map[string]any)["port"].(json.Number)
	if deepPort.String() != "8000" {
		t.Fatalf("deep handler port = %q, want %q", deepPort.String(), "8000")
	}

	// handlers[$.first] (without parentheses) is not a built-in call — it is
	// parsed as a plain Key scalar with the literal key name "$.first", which
	// does not match any field in the string elements of the handlers array and
	// must therefore return an error.
	badPath := "*pkg:$?var=services[name:default-name-proxy].handlers[category:main].outbounds[$.first()].handlers[$.first]"
	if _, err = noPerfMycelium.Spore(badPath); err == nil {
		t.Fatal("Spore(handlers[$.first] without parens): expected error, got nil")
	}
}

// noPerfMycelium is a test helper that loads noPerfection.json fresh each time
// so mutation tests start from a clean copy.
func loadNoPerfection(t *testing.T) *Mycelium {
	t.Helper()
	raw, err := os.ReadFile("noPerfection.json")
	if err != nil {
		t.Fatalf("ReadFile(noPerfection.json): %v", err)
	}
	m, err := Digest("pkg:json$#noPerfection.json", string(raw))
	if err != nil {
		t.Fatalf("Digest(noPerfection.json): %v", err)
	}
	return m
}

func TestInoculateService(t *testing.T) {
	m := loadNoPerfection(t)

	// Replace the first service (index 0) with a new map.
	newFirst := map[string]any{"type": "Independent", "name": "replaced-first"}
	if err := m.Inoculate("pkg:$?var=services[0]", newFirst); err != nil {
		t.Fatalf("Inoculate(services[0]): %v", err)
	}
	got, err := m.Spore("pkg:$?*var=services[0]")
	if err != nil {
		t.Fatalf("Spore after Inoculate first: %v", err)
	}
	svc, ok := got.(map[string]any)
	if !ok || svc["name"] != "replaced-first" {
		t.Fatalf("services[0].name = %v, want %q", svc["name"], "replaced-first")
	}

	// Replace the last service (index 2) with a different map.
	newLast := map[string]any{"type": "Independent", "name": "replaced-last"}
	if err := m.Inoculate("pkg:$?var=services[2]", newLast); err != nil {
		t.Fatalf("Inoculate(services[2]): %v", err)
	}
	got, err = m.Spore("pkg:$?*var=services[2]")
	if err != nil {
		t.Fatalf("Spore after Inoculate last: %v", err)
	}
	svc, ok = got.(map[string]any)
	if !ok || svc["name"] != "replaced-last" {
		t.Fatalf("services[2].name = %v, want %q", svc["name"], "replaced-last")
	}

	// Other services are unchanged.
	got, err = m.Spore("pkg:$?*var=services[1]")
	if err != nil {
		t.Fatalf("Spore(services[1]) after inoculate: %v", err)
	}
	mid, ok := got.(map[string]any)
	if !ok || mid["name"] != "entrypoint" {
		t.Fatalf("services[1].name = %v, want %q (unchanged)", mid["name"], "entrypoint")
	}
}

func TestInoculateOutboundHandlers(t *testing.T) {
	m := loadNoPerfection(t)

	// Overwrite the handlers array of the outbound named "hello-world" inside
	// services[name:default-name-proxy].handlers[category:main].
	newHandlers := []any{
		map[string]any{
			"type":     "Replier",
			"category": "main",
			"endpoint": map[string]any{"id": "localhost", "port": json.Number("9999")},
		},
	}
	path := "pkg:$?var=services[name:default-name-proxy].handlers[category:main].outbounds[name:hello-world].handlers"
	if err := m.Inoculate(path, newHandlers); err != nil {
		t.Fatalf("Inoculate(outbound handlers): %v", err)
	}

	// Spore confirms the handlers were replaced.
	got, err := m.Spore("pkg:$?*var=" + path[len("pkg:$?var="):])
	if err != nil {
		t.Fatalf("Spore(outbound handlers after inoculate): %v", err)
	}
	handlers, ok := got.([]any)
	if !ok || len(handlers) != 1 {
		t.Fatalf("handlers = %T (len %d), want []any of length 1", got, len(handlers))
	}
	h, ok := handlers[0].(map[string]any)
	if !ok || h["type"] != "Replier" || h["category"] != "main" {
		t.Fatalf("handler = %v, want {type:Replier, category:main}", h)
	}
	ep, ok := h["endpoint"].(map[string]any)
	if !ok {
		t.Fatalf("endpoint = %T, want map", h["endpoint"])
	}
	if ep["port"].(json.Number).String() != "9999" {
		t.Fatalf("endpoint port = %v, want 9999", ep["port"])
	}
}

func TestGraftService(t *testing.T) {
	m := loadNoPerfection(t)

	newSvc := map[string]any{"type": "Independent", "name": "grafted-service"}
	if err := m.Graft("pkg:$?var=services", newSvc); err != nil {
		t.Fatalf("Graft(services): %v", err)
	}

	got, err := m.Spore("pkg:$?*var=services")
	if err != nil {
		t.Fatalf("Spore(services) after graft: %v", err)
	}
	services, ok := got.([]any)
	if !ok || len(services) != 4 {
		t.Fatalf("len(services) = %d, want 4 after graft", len(services))
	}
	last, ok := services[3].(map[string]any)
	if !ok || last["name"] != "grafted-service" {
		t.Fatalf("services[3].name = %v, want %q", last["name"], "grafted-service")
	}
}

func TestPruneService(t *testing.T) {
	m := loadNoPerfection(t)

	if err := m.Prune("pkg:$?var=services[name:entrypoint]"); err != nil {
		t.Fatalf("Prune(services[name:entrypoint]): %v", err)
	}

	got, err := m.Spore("pkg:$?*var=services")
	if err != nil {
		t.Fatalf("Spore(services) after prune: %v", err)
	}
	services, ok := got.([]any)
	if !ok || len(services) != 2 {
		t.Fatalf("len(services) = %d, want 2 after prune", len(services))
	}
	for i, s := range services {
		obj := s.(map[string]any)
		if obj["name"] == "entrypoint" {
			t.Fatalf("services[%d] still has name %q after prune", i, "entrypoint")
		}
	}
}

func TestGraftAndPruneOutbound(t *testing.T) {
	m := loadNoPerfection(t)

	outboundPath := "pkg:$?var=services[name:default-name-proxy].handlers[category:main].outbounds"

	// Graft a new outbound.
	newOutbound := map[string]any{
		"type":     "Independent",
		"name":     "new-outbound",
		"handlers": []any{},
	}
	if err := m.Graft(outboundPath, newOutbound); err != nil {
		t.Fatalf("Graft(outbounds): %v", err)
	}

	got, err := m.Spore("pkg:$?*var=" + outboundPath[len("pkg:$?var="):])
	if err != nil {
		t.Fatalf("Spore(outbounds) after graft: %v", err)
	}
	outbounds, ok := got.([]any)
	if !ok || len(outbounds) != 2 {
		t.Fatalf("len(outbounds) = %d, want 2 after graft", len(outbounds))
	}
	added, ok := outbounds[1].(map[string]any)
	if !ok || added["name"] != "new-outbound" {
		t.Fatalf("outbounds[1].name = %v, want %q", added["name"], "new-outbound")
	}

	// Prune the original hello-world outbound.
	if err := m.Prune(outboundPath + "[name:hello-world]"); err != nil {
		t.Fatalf("Prune(outbounds[name:hello-world]): %v", err)
	}

	got, err = m.Spore("pkg:$?*var=" + outboundPath[len("pkg:$?var="):])
	if err != nil {
		t.Fatalf("Spore(outbounds) after prune: %v", err)
	}
	outbounds, ok = got.([]any)
	if !ok || len(outbounds) != 1 {
		t.Fatalf("len(outbounds) = %d, want 1 after prune", len(outbounds))
	}
	remaining, ok := outbounds[0].(map[string]any)
	if !ok || remaining["name"] != "new-outbound" {
		t.Fatalf("remaining outbound name = %v, want %q", remaining["name"], "new-outbound")
	}
}

