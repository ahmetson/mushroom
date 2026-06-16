# Mushroom

Mushroom is a new form of URL, as opposite to the web.

The web's core is a link. Links allow your computer to browse other computers
around the world. Along with the link comes the browser that implements the
linking functionality. The web also defines a document format, HTML, that
formalizes rendering. Then it defines a protocol that tells computers how to
interact. Lastly, it adds domain names to easily identify remote machines.

MushroomURL is the opposite. It adds dereferences. When your document has a
link, you can browse it. When your document has `*link`, it is a dereference
URL. The browser will embed its value right in the document.

Besides the opposite nature of the URLs, mushrooms are also intended for code,
whereas web links are intended for data.

## Schema

Mushroom URLs are a slightly edited form of
[Package URLs](https://github.com/package-url/purl-spec), so they stay
compatible with existing PURL libraries.

A Mushroom URL has the following shape:

```text
pkg:<type>/<package-id>#<module-id>?<resource-kind>=<resource-expression>&<other-key>=<other-value>
```

After the first `$` in a Mushroom URL, the parser treats the following text as a
regex expression that skips everything until the next Mushroom URL section. That
section starts at `#` for the module id, or at `?` for the resource part.

```text
pkg:json$#my-app-config.json
pkg:$?var=services[0]
```

In the first example, `$` skips from the package type until `#my-app-config.json`.
In the second, `$` skips until `?var=services[0]`.

This indirectly gives programming languages, frameworks, and ecosystems a shared
terminology. Instead of each language naming code elements differently, Mushroom
URLs describe them as resources attached to a package and module.

Resources are key-value parameters. The resource kind can only be one of:

- `var`: scalar values, constants, mutable values, and multi-value data.
- `func`: code that can be evaluated, including functions, code pieces, and methods.
- `object`: a collection of vars, state, and functions, such as a type with traits,
  a struct with methods, or an object with properties and functions.

Parentheses, `()`, mark scoped evaluation. When the parser encounters them, it
tries to evaluate that part of the resource expression.

```text
pkg:$?func=fooBar
```

This is incorrect because it identifies a function-like resource without an
evaluation scope.

```text
pkg:$?func=fooBar()
```

This is correct because `fooBar()` marks the function evaluation scope.

Examples:

```text
pkg:golang/github.com/ahmetson/hello-world#main?func=Greeting()
pkg:npm/@ahmetson/hello-world#src/index.js?func=greeting()
pkg:pypi/hello-world#hello_world/main.py?func=greeting()
```

By default, these are links. If you add `*` before a resource, the parser
evaluates that resource and returns its value instead.

```text
pkg:$?func=fooBar()
```

This is a link. It identifies the resource but does not evaluate it.

```text
pkg:$?*func=fooBar()
```

This evaluates `fooBar()` and returns the function result.

The `*` can also be placed before the `pkg:` prefix. In that form, it
dereferences the first resource in the URL.

```text
*pkg:golang/github.com/ahmetson/hello-world#main?func=greeting()
pkg:golang/github.com/ahmetson/hello-world#main?*func=greeting()
```

Both examples return the same value:

```text
"Hello world"
```

When `*` is placed before a module name, it marks the module for lazy loading.
The module is imported into memory, but its resource is not evaluated yet.

```text
pkg:golang/github.com/ahmetson/hello-world#*main?func=greeting()
```

Ready to start? Check out the tutorials below.

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
Digest(myceliumURL, data) -> mycelium
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

This repository includes a built-in JSON substrate. It receives a string and
returns a JSON mycelium.

```go
mycelium := substrate.Digest("pkg:json$#my-app-config.json")
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

- `soil.Germinate()` calls the substrate and digests the data at the given
  value.
- `soil.Recognize()` detects which mycelium the given path belongs to. If none
  is found, it returns the substrate that can handle it. If there is no
  mycelium and no substrate, it returns an error as the third parameter.
- `soil.Colony()` returns the list of myceliums in the colony.
- `soil.Substrates()` returns the list of engines that can create substrates.
- `soil.Hypha()` turns a string path into an absolute Mushroom URL. It is the
  parsing engine for the given string.

## Development

Run the test suite:

```sh
go test ./...
```

## License

This project is released under the public domain via CC0 1.0 Universal.
See [LICENSE](LICENSE).
