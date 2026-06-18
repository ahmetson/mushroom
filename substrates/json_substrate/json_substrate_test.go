package json_substrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ahmetson/mushroom"
)

func TestSubstrateImplementsMushroomSubstrate(t *testing.T) {
	var _ mushroom.Substrate = (*Substrate)(nil)
}

// digest is a test helper that digests inline data without touching the file
// system, mirroring the old standalone Root function but accepting raw data.
func digest(url string, data string) (*Mycelium, error) {
	soil := &mushroom.Soil{}
	pattern, err := soil.Hypha("pkg:json/$#$.json")
	if err != nil {
		return nil, err
	}
	substrate := &Substrate{url: pattern}
	if err := soil.AddSubstrate(substrate); err != nil {
		return nil, err
	}
	hypha, err := soil.Hypha(url)
	if err != nil {
		return nil, err
	}
	got, err := substrate.Digest(hypha, data, soil)
	if err != nil {
		return nil, err
	}
	soil.AddColony(got)
	return got.(*Mycelium), nil
}

func TestMyceliumImplementsMushroomMycelium(t *testing.T) {
	var _ mushroom.Mycelium = (*Mycelium)(nil)
}

func TestDigestAcceptsString(t *testing.T) {
	mycelium, err := digest("pkg:json$#my-app-config.json", `{"port":8080}`)
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
	pattern, _ := soil.Hypha("pkg:json/$#$.json")
	substrate := &Substrate{url: pattern}

	reqHypha, _ := soil.Hypha("pkg:json$#my-app-config.json")
	if _, err := substrate.Digest(reqHypha, []byte(`{"port":8080}`), soil); err == nil {
		t.Fatal("Digest returned nil error, want unsupported bytes error")
	}
}

func TestDigestUsesProvidedSoil(t *testing.T) {
	soil := &mushroom.Soil{}
	pattern, _ := soil.Hypha("pkg:json/$#$.json")
	substrate := &Substrate{url: pattern}

	reqHypha, _ := soil.Hypha("pkg:json$#my-app-config.json")
	got, err := substrate.Digest(reqHypha, `{"port":8080}`, soil)
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
	mycelium, err := digest("pkg:json$#my-app-config.json", `{"port":8080}`)
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
	mycelium, err := digest("pkg:json$#my-app-config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	// After Digest the mycelium is registered as a colony in the soil, so
	// Recognize may return either a colony or a substrate — both indicate the
	// JSON pattern is known to the soil.
	rh, _ := mycelium.Soil().Hypha("pkg:json/json_dir#my-app-config.json")
	colony, substrate, err := mycelium.Soil().Recognize(rh)
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}
	if colony == nil && substrate == nil {
		t.Fatal("Recognize found neither colony nor substrate")
	}
}

func TestJSONSubstratePatternRejectsOtherJSONLikeTypes(t *testing.T) {
	mycelium, err := digest("pkg:json$#my-app-config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	rh, _ := mycelium.Soil().Hypha("pkg:json-ad/file_dir#config.json")
	if _, _, err := mycelium.Soil().Recognize(rh); err == nil {
		t.Fatal("Recognize returned nil error, want json-ad to fail")
	}
}

func TestSporeReturnsJSONValue(t *testing.T) {
	mycelium, err := digest("pkg:json$#config.json", `{"services":[{"hostname":"localhost","port":8080}]}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("*pkg:$?var=services[0].port")
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("*pkg:$?var=port")
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	_, err = mycelium.Spore("pkg:$?var=port")
	if err == nil {
		t.Fatal("Spore returned nil error for link URL, want error")
	}
}

func TestSporeRejectsModuleDereference(t *testing.T) {
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	_, err = mycelium.Spore("pkg:json$#*config.json")
	if err == nil {
		t.Fatal("Spore returned nil error for module dereference, want error")
	}
}

func TestSporeTraversesArrayRootNameKey(t *testing.T) {
	mycelium, err := digest("pkg:json$#config.json", `[{"name":"alpha","key":"match"},{"name":"beta"}]`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("*pkg:$?var=name[key]")
	if err != nil {
		t.Fatalf("Spore returned error: %v", err)
	}
	if got != "alpha" {
		t.Fatalf("Spore returned %v, want %q", got, "alpha")
	}
}

func TestSporeTraversesNestedPath(t *testing.T) {
	mycelium, err := digest("pkg:json$#config.json", `{"outer":{"inner":{"port":3000}}}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("*pkg:$?var=outer.inner.port")
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
	// the same result as Spore("*pkg:$?var=port").
	mycelium, err := digest("pkg:json$#config.json", `{"fieldName":"port","port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("*pkg:$?var=(*pkg:$?var=fieldName)")
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
	mycelium, err := digest("pkg:json$#config.json",
		`{"key":"foo","services":[{"name":"foo","port":9000},{"name":"bar","port":9001}]}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	got, err := mycelium.Spore("*pkg:$?var=services[name:(*pkg:$?var=key)]")
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
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
	mycelium, err := digest("pkg:json$#config.json", `{
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
	if err != nil {
		t.Fatalf("Digest returned error: %v", err)
	}

	if _, err := mycelium.Link("*pkg:$?var=port"); err == nil {
		t.Fatal("Link returned nil error, want dereference error")
	}
}

func TestDigestRejectsNonMushroomURL(t *testing.T) {
	if _, err := digest("config.json", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want invalid URL error")
	}
}

func TestDigestRejectsDereferenceURL(t *testing.T) {
	if _, err := digest("*pkg:json$#config.json", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want dereference URL error")
	}
}

func TestDigestRejectsNonJSONType(t *testing.T) {
	if _, err := digest("pkg:golang$#config.json", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want invalid type error")
	}
}

func TestDigestRejectsNonJSONModuleID(t *testing.T) {
	if _, err := digest("pkg:json$#config.txt", `{"port":8080}`); err == nil {
		t.Fatal("Digest returned nil error, want invalid module id error")
	}
}

func TestLinkFillsTypeAndModuleFromMyceliumURL(t *testing.T) {
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
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
	mycelium, err := digest("pkg:json$#config.json", `{"port":8080}`)
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
	noPerfMycelium, err := digest("pkg:json$#noPerfection.json", string(raw))
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}

	// Link to services and spore the full services array.
	if _, err := noPerfMycelium.Link("pkg:$?var=services"); err != nil {
		t.Fatalf("Link(services): %v", err)
	}
	got, err := noPerfMycelium.Spore("*pkg:$?var=services")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[$.first()]")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[$.last()]")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[0]")
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
	if _, err := noPerfMycelium.Spore("*pkg:$?var=services[3]"); err == nil {
		t.Fatal("Spore(services[3]) returned nil error, want out-of-range error")
	}

	// Numeric indexes and $.first() / $.last() require an array.
	// services[0].type is a plain string, so further indexing must fail.
	if _, err := noPerfMycelium.Spore("*pkg:$?var=services[0].type[0]"); err == nil {
		t.Fatal("Spore(services[0].type[0]) returned nil error, want non-array error")
	}
	if _, err := noPerfMycelium.Spore("*pkg:$?var=services[0].type[$.first()]"); err == nil {
		t.Fatal("Spore(services[0].type[$.first()]) returned nil error, want non-array error")
	}
	if _, err := noPerfMycelium.Spore("*pkg:$?var=services[0].type[$.last()]"); err == nil {
		t.Fatal("Spore(services[0].type[$.last()]) returned nil error, want non-array error")
	}

	// Filter by type:Proxy returns both proxy services.
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[type:Proxy]")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[type:Proxy][$.last()]")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[0].handlers[0].outbounds[0]")
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
	if _, err = noPerfMycelium.Spore("*pkg:$?var=services[(*pkg:$?var=services[0].name)]"); err == nil {
		t.Fatal("Spore(services[(*lambda)]): expected error — substituted key 'default-name-proxy' is not a field on any service")
	}

	// Non-dereference lambda in scalar position:
	// (pkg:$?var=services[0].name) resolves to the link string
	// "pkg:json$#noPerfection.json?var=services[0].name" and is substituted,
	// producing services[pkg:json$#noPerfection.json?var=services[0].name].
	// The colon makes it a KeyValue filter with Key="pkg" — no service has
	// that field, so Spore must fail.
	if _, err = noPerfMycelium.Spore("*pkg:$?var=services[(pkg:$?var=services[0].name)]"); err == nil {
		t.Fatal("Spore(services[(non-deref lambda)]): expected error — substituted link URL is not a valid filter")
	}

	// Plain key scalar [default-name-proxy] (no colon) looks for a field NAMED
	// "default-name-proxy" on each service object — it is NOT a name-value filter.
	// Since no service has a field with that literal name, Spore must fail.
	if _, err = noPerfMycelium.Spore("*pkg:$?var=services[default-name-proxy]"); err == nil {
		t.Fatal("Spore(services[default-name-proxy]): expected error — plain key looks for a field named 'default-name-proxy', not a service with that name")
	}

	// The correct way to filter by name value is the key:value form.
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[name:default-name-proxy]")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[name:(*pkg:$?var=services[0].name)]")
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
	if _, err = noPerfMycelium.Spore("*pkg:$?var=services[name:default-name-proxy.first()]"); err == nil {
		t.Fatal("Spore(services[name:default-name-proxy.first()]): expected error — value is a literal string, not a call")
	}

	// services[name:default-name-proxy].first() — first() is a SEPARATE segment
	// after the bracket. The filter returns []any with one matching service, then
	// first() picks its first element, yielding the service map directly.
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[name:default-name-proxy].first()")
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
	got, err = noPerfMycelium.Spore("*pkg:$?var=services[$.first()]")
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
	m, err := digest("pkg:json$#noPerfection.json", string(raw))
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
	got, err := m.Spore("*pkg:$?var=services[0]")
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
	got, err = m.Spore("*pkg:$?var=services[2]")
	if err != nil {
		t.Fatalf("Spore after Inoculate last: %v", err)
	}
	svc, ok = got.(map[string]any)
	if !ok || svc["name"] != "replaced-last" {
		t.Fatalf("services[2].name = %v, want %q", svc["name"], "replaced-last")
	}

	// Other services are unchanged.
	got, err = m.Spore("*pkg:$?var=services[1]")
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
	got, err := m.Spore("*pkg:$?var=" + path[len("pkg:$?var="):])
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

	got, err := m.Spore("*pkg:$?var=services")
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

	got, err := m.Spore("*pkg:$?var=services")
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

	got, err := m.Spore("*pkg:$?var=" + outboundPath[len("pkg:$?var="):])
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

	got, err = m.Spore("*pkg:$?var=" + outboundPath[len("pkg:$?var="):])
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

// TestSubstrateForageAndSow verifies the full Forage → Sow → Forage round-trip.
//
//  1. Sow writes a JSON file into a temp directory.
//  2. Forage reads it back and returns the raw string.
//  3. The round-tripped content decodes to the same data.
//
// It also confirms that a URL which does not satisfy the substrate pattern is
// rejected by both Forage and Sow.
func TestSubstrateForageAndSow(t *testing.T) {
	dir := t.TempDir()

	soil := &mushroom.Soil{}
	pattern, _ := soil.Hypha("pkg:json/$#$.json")
	s := &Substrate{url: pattern}

	// Build a Hypha pointing at dir/data.json.  We construct it directly so
	// that we can supply the absolute temp-dir path as PackageID.
	fileURL := mushroom.Hypha{
		URL:       true,
		Type:      "json",
		PackageID: dir,
		ModuleID:  "data.json",
	}

	original := map[string]any{"port": float64(8080), "name": "test-app"}

	// Sow the map (non-string data path – marshalled to JSON).
	if err := s.Sow(fileURL, original); err != nil {
		t.Fatalf("Sow: %v", err)
	}

	// Confirm the file was actually written.
	if _, err := os.Stat(filepath.Join(dir, "data.json")); err != nil {
		t.Fatalf("file not found after Sow: %v", err)
	}

	// Forage reads it back.
	raw, err := s.Forage(fileURL)
	if err != nil {
		t.Fatalf("Forage: %v", err)
	}
	rawStr, ok := raw.(string)
	if !ok {
		t.Fatalf("Forage returned %T, want string", raw)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(rawStr), &decoded); err != nil {
		t.Fatalf("unmarshal Forage result: %v", err)
	}
	if decoded["port"] != float64(8080) || decoded["name"] != "test-app" {
		t.Fatalf("round-trip data mismatch: got %v", decoded)
	}

	// Sow with a pre-marshalled string must also round-trip correctly.
	rawJSON := `{"version":"2.0"}`
	if err := s.Sow(fileURL, rawJSON); err != nil {
		t.Fatalf("Sow(string): %v", err)
	}
	raw2, err := s.Forage(fileURL)
	if err != nil {
		t.Fatalf("Forage after string Sow: %v", err)
	}
	if raw2.(string) != rawJSON {
		t.Fatalf("string round-trip: got %q, want %q", raw2.(string), rawJSON)
	}

	// A URL that does not satisfy the substrate pattern must be rejected.
	badURL := mushroom.Hypha{
		URL:       true,
		Type:      "yaml",
		PackageID: dir,
		ModuleID:  "data.yaml",
	}
	if _, err := s.Forage(badURL); err == nil {
		t.Fatal("Forage(badURL): expected error, got nil")
	}
	if err := s.Sow(badURL, "{}"); err == nil {
		t.Fatal("Sow(badURL): expected error, got nil")
	}
}

// TestHyphaModuleLambda validates that:
//   - A plain URL is returned as a Hypha without any colony evaluation.
//   - A lambda in the module position is evaluated, but the root Hypha is not.
//   - The same holds when the root Hypha is itself a dereference URL.
func TestHyphaModuleLambda(t *testing.T) {
	colony, err := digest("pkg:json$#config.json", `{"config-name":"config.json"}`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	soil := colony.Soil()

	// Scenario 1: pkg:json#config.json?var=name
	//
	// No lambdas present. Hypha() returns the parsed URL without touching any
	// colony (Link / Spore are never called).
	t.Run("plain URL returned without link evaluation", func(t *testing.T) {
		h, err := soil.Hypha("pkg:json#config.json?var=name")
		if err != nil {
			t.Fatalf("Hypha error: %v", err)
		}
		if h.Type != "json" {
			t.Fatalf("Type = %q, want %q", h.Type, "json")
		}
		if h.ModuleID != "config.json" {
			t.Fatalf("ModuleID = %q, want %q", h.ModuleID, "config.json")
		}
		if h.ResourceKind != mushroom.ResourceKindVar {
			t.Fatalf("ResourceKind = %q, want %q", h.ResourceKind, mushroom.ResourceKindVar)
		}
		if h.ResourcePath.Raw != "name" {
			t.Fatalf("ResourcePath = %q, want %q", h.ResourcePath.Raw, "name")
		}
		if h.Dereference {
			t.Fatal("Dereference = true, want false")
		}
	})

	// Scenario 2: pkg:json#(*pkg:$?var=config-name)  (defaults = colony.url)
	//
	// The `(` follows `#`, which is not an identifier character, so it is a
	// lambda.  The inner dereference *pkg:$?var=config-name is filled from
	// defaults → *pkg:json$#config.json?var=config-name, which Spore resolves
	// to "config.json".
	//
	// Resolved path: "pkg:json#config.json"
	// Root Hypha is NOT a dereference — it is simply returned, not executed.
	t.Run("module lambda evaluates, root hypha is not executed", func(t *testing.T) {
		h, err := soil.Hypha("pkg:json#(*pkg:$?var=config-name)", colony.url)
		if err != nil {
			t.Fatalf("Hypha error: %v", err)
		}
		if h.Type != "json" {
			t.Fatalf("Type = %q, want %q", h.Type, "json")
		}
		if h.ModuleID != "config.json" {
			t.Fatalf("ModuleID = %q, want %q", h.ModuleID, "config.json")
		}
		if h.Dereference {
			t.Fatal("root Hypha must not be a dereference")
		}
	})

	// Scenario 3: *pkg:json#(*pkg:$?var=config-name)  (defaults = colony.url)
	//
	// Same module lambda as scenario 2, but the root URL is prefixed with `*`.
	// After the lambda is evaluated the resolved path is "*pkg:json#config.json".
	// Hypha() returns a dereference Hypha — it is not Spored or Linked itself.
	t.Run("dereference root with module lambda is parsed but not executed", func(t *testing.T) {
		h, err := soil.Hypha("*pkg:json#(*pkg:$?var=config-name)", colony.url)
		if err != nil {
			t.Fatalf("Hypha error: %v", err)
		}
		if h.Type != "json" {
			t.Fatalf("Type = %q, want %q", h.Type, "json")
		}
		if h.ModuleID != "config.json" {
			t.Fatalf("ModuleID = %q, want %q", h.ModuleID, "config.json")
		}
		if !h.Dereference {
			t.Fatal("root Hypha must be a dereference URL")
		}
	})
}

// TestHyphaLambdaComposition validates four lambda-embedding scenarios that
// exercise how Soil.Hypha resolves lambdas during URL construction.
//
// A "fake colony" mycelium is created from a JSON object that stores two
// string variables:
//
//	package-name        = "$"           (a wildcard package identifier)
//	package-and-module  = "$#config.json" (wildcard package + concrete module)
func TestHyphaLambdaComposition(t *testing.T) {
	colony, err := digest("pkg:json$#config.json",
		`{"package-name":"$","package-and-module":"$#config.json"}`)
	if err != nil {
		t.Fatalf("setup colony: %v", err)
	}
	soil := colony.Soil()

	// --- Scenario 1 --------------------------------------------------------
	// Path: (*)(pkg:(packageormodule))
	//
	// (*)                       → plain symbol "*"
	// (pkg:(packageormodule))   → inner (packageormodule) is a plain symbol
	//                             → "pkg:packageormodule" (non-dereference link,
	//                               no colony for type "packageormodule")
	//
	// Resolved path: "*pkg:packageormodule"
	// Parsed Hypha:  Dereference=true, Type="packageormodule"
	// Usage fails:   no colony recognises type "packageormodule".
	t.Run("symbol-star plus link produces dereference with wrong type", func(t *testing.T) {
		h, err := soil.Hypha("(*)(pkg:(packageormodule))")
		if err != nil {
			t.Fatalf("Hypha returned error: %v", err)
		}
		if !h.Dereference {
			t.Fatal("Dereference = false, want true")
		}
		if h.Type != "packageormodule" {
			t.Fatalf("Type = %q, want %q", h.Type, "packageormodule")
		}
		// Recognize fails: no substrate or colony has type "packageormodule".
		if _, _, err := soil.Recognize(h); err == nil {
			t.Fatalf("Recognize(%q): want error, got nil", h.String())
		}
	})

	// --- Scenario 2 --------------------------------------------------------
	// Path: pkg:(pkg:json#config.json?var=package-name)
	//
	// The lambda (pkg:json#config.json?var=package-name) is NON-dereference.
	// evalLambda calls colony.Link → returns the absolute link string
	// "pkg:json$#config.json?var=package-name".
	//
	// That link string is embedded verbatim after "pkg:", producing:
	//   "pkg:pkg:json$#config.json?var=package-name"
	//
	// The parser sees type "pkg:json" (because the embedded "pkg:" prefix is
	// treated as part of the type token) — semantically invalid.
	// Recognize fails: no substrate has type "pkg:json".
	t.Run("non-dereference lambda embeds link literally and corrupts type", func(t *testing.T) {
		h, err := soil.Hypha("pkg:(pkg:json#config.json?var=package-name)")
		if err != nil {
			t.Fatalf("Hypha returned error: %v", err)
		}
		if h.Type != "pkg:json" {
			t.Fatalf("Type = %q, want %q (link embedded verbatim into type position)",
				h.Type, "pkg:json")
		}
		// Recognize fails: type "pkg:json" is not registered.
		if _, _, err := soil.Recognize(h); err == nil {
			t.Fatalf("Recognize(%q): want error, got nil", h.String())
		}
	})

	// --- Scenario 3 --------------------------------------------------------
	// Path: pkg:json/(*pkg:json#config.json?var=package-name)
	//
	// The lambda (*pkg:json#config.json?var=package-name) IS dereference.
	// evalLambda calls colony.Spore → value of "package-name" = "$".
	//
	// Resolved path: "pkg:json/$"
	// Parsed Hypha:  Type="json", PackageID="$", no module — valid, usable.
	t.Run("dereference lambda fetches value and builds valid URL", func(t *testing.T) {
		h, err := soil.Hypha("pkg:json/(*pkg:json#config.json?var=package-name)")
		if err != nil {
			t.Fatalf("Hypha returned error: %v", err)
		}
		if h.Type != "json" {
			t.Fatalf("Type = %q, want %q", h.Type, "json")
		}
		if h.PackageID != "$" {
			t.Fatalf("PackageID = %q, want %q", h.PackageID, "$")
		}
	})

	// --- Scenario 4 --------------------------------------------------------
	// Path: pkg:json/(*pkg:$?var=package-and-module)  (defaults = colony.url)
	//
	// The lambda has a wildcard type ($). With colony.url as defaults:
	//   *pkg:$?var=package-and-module → *pkg:json$#config.json?var=package-and-module
	// colony.Spore returns the value "$#config.json".
	//
	// Resolved path: "pkg:json/$#config.json"
	// Parsed Hypha:  Type="json", PackageID="$", ModuleID="config.json" — fully valid.
	// This shows how a single lambda can reconstruct a complete module URL
	// by dereferencing a stored "package-and-module" variable.
	t.Run("dereference wildcard lambda with defaults reconstructs full module URL", func(t *testing.T) {
		h, err := soil.Hypha("pkg:json/(*pkg:$?var=package-and-module)", colony.url)
		if err != nil {
			t.Fatalf("Hypha returned error: %v", err)
		}
		if h.Type != "json" {
			t.Fatalf("Type = %q, want %q", h.Type, "json")
		}
		if h.PackageID != "$" {
			t.Fatalf("PackageID = %q, want %q", h.PackageID, "$")
		}
		if h.ModuleID != "config.json" {
			t.Fatalf("ModuleID = %q, want %q", h.ModuleID, "config.json")
		}
	})
}

