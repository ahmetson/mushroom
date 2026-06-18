# MushroomURL

MushroomURL is the opposite of the Web URL. When your document has a web URL, you can visit it via a browser. 

Mushroom URLs add a dereferences with the: `*link`.  The browser will dereference `*` and embed its value right in the document. You don't have to browse the links.

Another important distinction is, mushroom URLs intended for code, whereas web URLs are intended for data.

## Schema

Mushroom URLs are a slightly edited form of [Package URLs](https://github.com/package-url/purl-spec).

A Mushroom URL schema:

```text
pkg:type/package#module?resource-kind=resource-path&key=value
```

This is a link. To turn it into a dereference, we need to add a `*`. Dereference could be for
`module`, or for `resource-kind`. To make code readiable resource dereferences can be put at the front of the URLs.

Module derefence schema indicates, lazy load of the module but not evaluate any data yet:

`pkg:type/package#***module**?resource-kind=resource-path&key=value`.
You just add the `*` dereference operator in the beginning of module.

The dereference of the resources can be in a two way:

`pkg:type/package#module?***resource-kind**=resource-path&key=value`.
`***pkg**:type/package#module?resource-kind=resource-path&key=value`.

You just add the `*` dereference operator in the beginning of the Mushrom URL or before `resource-kind`.

### UTF-8 And Trimming

Mushroom assumes strings are UTF-8. Before parsing a URL, it removes spaces,
whitespace, and invisible UTF characters, so all traversed data is handled in a
trimmed form:

`pkg: type/package # module` is identical to `pkg:type/package#module`.

### Symbols and URL detections

Mushroom URLs are symbolic. So any value is treated are valid, and treated as a string.

If the string after trimming starts with `pkg:` or `*pkg:` then the entire line is treated as a MushroomURL.

### Regular expressions

For the urls, you don't have to provide all parts as an absolute URL.
You can omit the parts and work with some aspects only. Except the `pkg:` indicator of Mushroom URLs.

URLs are supporting a special regular expressions as well:

The first is `$`. It means it can be any text:

`pkg:$`matches to `pkg:json`.
`pkg:/$item` matches to `pkg:json/random-item`, it simply means package ends with `item`.

This is useful for pattern matching when you want to query the data but don't know want to write a full path.

Second type of regular expressions are lambdas defined as `()`.
If mushroom URL receives a lambda, first it evaluates it. Then hanldes the URL itself:

`pkg:json#(config.json)` is treated as `pkg:json#config.json`.
If the lambda has another mushroom url, its evaluates and puts its value first.
To put it we need to make it dereference so, otherwise it will put a link itself which is undefined in behavior.

`pkg:json#(*pkg:$?var=fileName)?var=portID` will see a lambda with dereferencing url, and evaluate it first:

`pkg:json#config.json?var=portID` and then evaluate the url itself.

### Types, Packages and Modules

They are identical or at least shall not break the [purl](https://github.com/package-url/purl-spec).

Indirectly mushroom URL gives programming languages, frameworks, and ecosystems a shared
terminology. Instead of each language naming code elements differently, Mushroom
URLs describe them as resources attached to a package and module.

What is package and module depends on the type of package managers. For example
in `go` programming language consists of modules which then consists of packages for each directory. In purl its reversed. Each directory is treated as a module, while `go.mod` is treated as a package.

Or in the nodejs each project is a package, and each file is a module. That is matching to the `purl` so it leaves as it is.

### Resources

Resources are key-value parameters. The resource kind can only be one of:

- `var`: scalar values, constants, mutable values, and multi-value data any named data in the module.
- `func`: code that can be evaluated, including functions, code pieces, and methods.
- `object`: a collection of vars, state, and functions, such as a type with traits,
a struct with methods, or an object with properties and functions.

The value after `=` is a resource path. A resource path
is built from path segments such as `path0`, `path0.path1`, or `path[arg]`.
If a path to object, var or function is talking to a multi-value data then use the `[]` brackets with the arguments for filtering out. A bracket argument can be:

- `[key: value]` filter the first element whose key matches the value
- a number indicating an index
- key if its map only.
- a call: `name(args)`
- $ regular expression indicates any.
- For keys and $, and key values you can add the built in calls:

```text
path0[key: $].path1[key1: ($.first())]
path1[item: hello world].path2[item: (hello world.first())]
```

Mushroom defines two built-in call functions: `first()` and `last()` that can be used in the scalars.

If the resource kind is `func`, the last path segment must be a call.

## Tutorials

### Working With A Mycelium

For mushrooms to embed and interact with data, they work with a network of
resources. That network is called a mycelium. In biology, mycelium is the name
for the network created by fungi in nature.

All interaction with Mushroom URLs goes through a mycelium network.

A mycelium is created from a substrate. Substrate is also a biological term from
fungiology: it describes the material where a mycelium network is created. In
Mushroom, a substrate converts data into a traversal network of resources.

Every substrate provides a digest function:

```text
Digest(myceliumURL, data, soil) -> mycelium
```

`Digest` converts the given data into a mycelium network. The resulting
mycelium combines the substrate and the document into one traversable resource
network.

Myceliums expose four core functions. Each function receives a Mushroom URL or
an evaluated Mushroom value.

- `mycelium.Link("your data")` returns a full path link and whether it is valid.
A link cannot contain a dereference.
- `mycelium.Spore("dereference")` evaluates a dereference URL and returns the
value at that path.
- `mycelium.Fruit(sporedValue)` traverses a value, finds any links inside it,
and evaluates those links when needed.
- `mycelium.Mineralize()` returns the data in its digested format.
- `mycelium.MushroomURL()` returns the absolute link to the mycelium.

This repository includes a built-in JSON substrate. It receives a string and
returns a JSON mycelium.

```go
soil := &mushroom.Soil{}
mycelium, err := substrate.Digest(soil.Hypha("pkg:json$#my-app-config.json"), fileContent, soil)
if err != nil {
	return err
}
```

Use `Link` to resolve a resource path without evaluating it:

```go
link, valid := mycelium.Link("pkg:$?var=services[0]")
```

This returns an absolute path to the first `services` element in the JSON
configuration, plus whether the link is valid.

Use `Spore` to evaluate a dereference and return the value:

```go
service := mycelium.Spore("pkg:$?*var=services[0]")
```

This returns the element at `services[0]`.

Use `Fruit` to traverse a spored value and embed any linked values it contains:

```go
service = mycelium.Fruit(service.String())
```

For example, if `services.moduleURL` is a Mushroom URL, `Fruit` will traverse it
and embed its result by using `mycelium.Spore`.

Use `Mineralize` to convert the mycelium back into the substrate format:

```go
jsonStr := mycelium.Mineralize()
```

Mineralize is also a scientific term. In soil science,
[mineralization](https://en.wikipedia.org/wiki/Mineralization_(soil_science))
basically means releasing organic matter into soil as nutrients.

For the JSON substrate, `Mineralize` returns a JSON string that can be saved
back into a file.

### Mycelium Of Myceliums

A single mycelium is one document after digesting a substrate with raw
nutrients. What if you need multiple JSON files, or other file types together?
That is handled by a soil, the root of Mushroom URLs.

Soil keeps path parsing. It also keeps a colony of myceliums. This layer is
usually internal and handled by myceliums and substrates, but it is worth
knowing when you want data to traverse across multiple resources.

For example, you may have a `go` programming language substrate that turns
source code into a mycelium. Then you want to return service parameters that
depend on a JSON config.

The mycelium, using the soil, can digest the path, create a mycelium, add it to
the colony, and whenever it receives a URL from any mycelium it can traverse
into any of them. If the package type is not supported by the soil, it returns
an error.

```go
endpoint = goSourceMycelium.Spore("pkg:$?*var=serviceEndpoint")
```

This returns the endpoint with its linked values still inside it:

```json
{
  "hostname": "localhost",
  "port": "*pkg:json$#absolute-file-path/file-name.json?var=port"
}
```

Then retrieve the final value with `Fruit`:

```go
endpoint = goSourceMycelium.Fruit(endpoint)
```

`Fruit` evaluates the linked `port` value and passes it into the endpoint.

Soil exposes these functions:

- `soil.AddSubstrate(substrate)` validates the substrate Mushroom URL as a link
and registers it by its Hypha pattern so `Recognize` can find it later. For
example, the JSON substrate registers `pkg:json/$#$.json`, where `$` matches
any package id and `$.json` matches any module id ending in `.json`.
- `soil.Germinate(hypha, substrate)` is the canonical way to load an
unrecognised mycelium on demand. It: registers the substrate if not already
present; forages the raw data for `hypha` from the substrate; digests it into
a new `Mycelium`; and registers that mycelium as a colony so future lookups
find it without foraging again. `Spore` calls this automatically when it
encounters an `ErrUnrecognizedMycelium`.
- `soil.Recognize()` detects which mycelium the given path belongs to. If none
is found, it returns the substrate that can handle it. If there is no
mycelium and no substrate, it returns an error as the third parameter.
- `soil.Colony()` returns the list of myceliums in the colony.
- `soil.Substrates()` returns the list of engines that can create substrates.
- `soil.Hypha()` turns a string path into an absolute Mushroom URL. It is the
parsing engine for the given string.

## Built in substrates: JSON

`substrates/json_substrate` digests the json string into a mycelium.
Then you can traverse and talk to the json elements using mushroom URLs.

The substrates Mushroom URL is `pkg:json/$#$.json`. It means module name must end with `.json` and type of package is `json`. Any mycelium you creat must satisfy the substrate's mushroom url.

Tutorial:

```go
import "github.com/ahmetson/mushroom/substrates/json_substrate"

mycelium, _ := json_substrate.Root(`pkg:json/./substrates/json_substrate/#noPerfection.json`)
handlerPath, _ := mycelium.Link(`pkg:$?var=services[$.first()].handlers[category: main]`) // get absolute path to main handler in noPerfection.json
handlerData, _ := mycelium.Spore("*" + handlerPath) //dereference the handler
handlerData, _ = mycelium.Fruit(handlerData) // If handler data has mushroom urls they are dereferenced in a recursive mannger. For example handlers.outbounds[0].handlers is a package url as well.
```

The `PackageID` and `ModuleID` in the Mushroom URL are not just conventions — the substrate can use them directly to read and write files on the file system via `Forage` and `Sow` (see below).

Json substrates also doesn't support `object` nor `func` resource kinds. Only variables.

### Reading and writing files with a substrate

`json_substrate.Substrate` implements the `mushroom.Substrate` interface's `Forage` and `Sow` methods.
`Forage` combines the `PackageID` and `ModuleID` of the provided Mushroom URL into a file path, reads the file, and returns its raw content as a string. That string can be passed directly to `Digest`. The read is protected by a read lock, so concurrent `Forage` calls are safe.

`Sow` is the inverse: it takes a Go value (`data any`) and writes it to the same derived file path. If `data` is already a string it is written verbatim; any other value is JSON-marshalled first. The write is protected by an exclusive lock, preventing races with concurrent `Forage` or `Sow` calls.

```go
import "github.com/ahmetson/mushroom/substrates/json_substrate"

// Root forages the file and digests it in one step.
mycelium, _ := json_substrate.Root("pkg:json/configs#app.json")

// Access the substrate from the mycelium for Forage / Sow.
substrate := (*mycelium.Substrate()).(*json_substrate.Substrate)
soil := mycelium.Soil()
configURL := soil.Hypha("pkg:json/configs#app.json")

// Mutate in memory, then persist back to disk.
mycelium.Inoculate("pkg:$?var=port", 9090)
updated, _ := mycelium.Mineralize()
substrate.Sow(configURL, updated)
```

### Mutating a JSON mycelium

`json_substrate.Mycelium` exposes three extra methods — not part of the core
`mushroom.Mycelium` interface — for in-place mutation:

`**Inoculate(path, value)**` — overwrites the value at the given resource path.

```go
// Replace the first service entirely.
mycelium.Inoculate("pkg:$?var=services[0]", newServiceMap)

// Overwrite a single field on a named service.
mycelium.Inoculate("pkg:$?var=services[name:foo].port", 9001)

// Replace the handlers array of a specific outbound.
mycelium.Inoculate(
    "pkg:$?var=services[name:foo].handlers[category:main].outbounds[name:bar].handlers",
    newHandlers,
)
```

`**Graft(path, item)**` — appends an item to the array at path.

```go
// Add a new service to the end of the services array.
mycelium.Graft("pkg:$?var=services", newServiceMap)

// Add a new outbound to a specific handler.
mycelium.Graft(
    "pkg:$?var=services[name:foo].handlers[category:main].outbounds",
    newOutbound,
)
```

`**Prune(path)**` — removes all items matching the filter from their parent array.

```go
// Remove a service by name.
mycelium.Prune("pkg:$?var=services[name:foo]")

// Remove a specific outbound.
mycelium.Prune(
    "pkg:$?var=services[name:foo].handlers[category:main].outbounds[name:bar]",
)
```

All three mutations are **in-memory only**. The `mushroom.Mycelium` interface is
not changed — the methods are specific to `*json_substrate.Mycelium`.
After mutating, call `Mineralize` to get the updated JSON string and handle
persistence yourself:

```go
updated, _ := mycelium.Mineralize()   // returns string
os.WriteFile("config.json", []byte(updated.(string)), 0o644)
```

Because the mutations update the same in-memory tree that `Spore`, `Link`, and
`Fruit` read from, subsequent calls to those methods immediately reflect the
changes — no need to re-digest.

## Development

Run the test suite:

```sh
go test ./...
```

## License

This project is released under the public domain via CC0 1.0 Universal.
See [LICENSE](LICENSE).