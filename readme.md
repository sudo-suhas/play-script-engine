# play-script-engine

Analysis of scripting engines for Meteor 'mapping' processor. See
[odpf/meteor#420][odpf-meteor-issues-420].

## Requirements

The current API contract of processor would apply here as well. So it would
accept a `models.Record` and return a `models.Record`. The user should be able
to write a script that gets access to the [`Asset`][odpf-proton-asset-proto] and
modify the same. The script env should support:

- Run as a black box with capability to detect and recover from errors.
- Assigning a computed or literal value to a field
- Iterating a list and transforming each element in the list
- Basic string manipulation functions such as replace, join, split, lowercase,
  uppercase etc.
- Conditionals such as if, case etc with boolean logic and arithmetic
- Custom helper functions that could be added in the future. ex: resolve DNS for
  IP.

Additionally, it is preferred if the Go type information is retained while
running the script.

## Scripting Engines

With the script, we try to do the following:

- Add a label to the asset - `"script_engine": "<current_script_engine>`.
- Add a label to each entity. Ex: `"catch_phrase": "..."`.
- Set an EntityName for each feature based on the following mapping:
  - `ongoing_placed_and_waiting_acceptance_orders: customer_orders`
  - `ongoing_orders: customer_orders`
  - `merchant_avg_dispatch_arrival_time_10m: merchant_driver`
  - `ongoing_accepted_orders: merchant_orders`
- Set the owner as `{Name: Big Mom, Email: big.mom@wholecakeisland.com}`.
- Set the `Url` using a function that is passed in.
- For each lineage upstream, if the service is Kafka, apply a string replace op
  on the URN - `{.yonkou.io => }`.

The following scripting engines are considered:

1. [otto](#otto)
2. [Goja](#goja)
3. [Bloblang](#bloblang)
4. [go-lua](#go-lua)
5. [GopherLua](#gopherlua)
6. [Tengo](#tengo)
7. [Anko](#anko)
8. [gojq](#gojq)

### otto

[GitHub - robertkrimen/otto: A JavaScript interpreter in Go (golang)][otto]

#### Sample Script

```js
asset.labels = _.extend({script_engine: 'otto'}, asset.labels);

_.each(data.entities, function (e) {
    e.labels = _.extend({catch_phrase: 'I\'ll be back'}, e.labels);
});

_.each(data.features, function (f) {
    switch (f.name) {
        case 'ongoing_placed_and_waiting_acceptance_orders':
        case 'ongoing_orders':
            f.EntityName = 'customer_orders';
            break;
        case 'merchant_avg_dispatch_arrival_time_10m':
            f.EntityName = 'merchant_driver';
            break;
        case 'ongoing_accepted_orders':
            f.EntityName = 'merchant_orders';
            break;
    }
})

asset.owners = [{name: "Big Mom", email: "big.mom@wholecakeisland.com"}]
    .concat(asset.owners);

asset.url = urler(asset.name);

_.chain(asset.lineage.upstreams)
    .filter(function (u) {
        return u.service === "kafka";
    })
    .each(function (u) {
        u.urn = u.urn.replace('.yonkou.io', '');
    });
```

[`otto/otto_transform.go`](./otto/otto_transform.go)

#### Pros

- Is able to translate Go type information and directly modify fields (with some
  caveats).
- Can optionally use the [underscore][underscore-js] library
  with [otto@`master/underscore`][otto-underscore].
- Very popular library with 6.9K stars.

#### Cons

- The type information for Data field works against us and cannot be directly
  modified. We need to assign a second global field after unmarshaling the field
  of type *anypb.Any. Furthermore, we access fields by the Go field names where
  proto field names would have been more appropriate.
- Is not able to detect and protect against non-existent fields (silently does
  nothing).
- Error handling is with panic. For ex: it can panic for cases such as
  assignment to a `nil` map.
- In some cases, unset/nil value needs to be initialised using Go field name.
  Accessing fields that are set can be done using field names starting with
  lowercase. For ex: If the property `entity_name` is unset, it cannot be set
  using `entityName`, need to use `EntityName`.
- No releases/tags for the library.
- No direct support for `context.Context`, a workaround is possible using an '
  interrupt' - [otto#halting-problem][otto-halting-problem].
- There are some known caveats - [otto#caveat-emptor][otto-caveat-emptor].
  Furthermore, the version of JavaScript is quite outdated so might feel jarring
  to people who do know modern JS.
- Not as actively maintainer based on the 122 open issues and 10 open pull
  requests.

### Goja

[GitHub - dop251/goja: ECMAScript/JavaScript engine in pure Go][goja]

#### Sample Script

```js
asset.labels = Object.assign({script_engine: 'goja'}, asset.labels);

for (const e of data.entities) {
    e.labels = Object.assign(
        {catch_phrase: 'Say hello to my little friend.'}, e.labels
    );
}

for (const f of data.features) {
    switch (f.name) {
        case 'ongoing_placed_and_waiting_acceptance_orders':
        case 'ongoing_orders':
            f.entity_name = 'customer_orders';
            break;
        case 'merchant_avg_dispatch_arrival_time_10m':
            f.entity_name = 'merchant_driver';
            break;
        case 'ongoing_accepted_orders':
            f.entity_name = 'merchant_orders';
            break;
    }
}

asset.owners = [{name: 'Big Mom', email: 'big.mom@wholecakeisland.com'}]
    .concat(asset.owners);

asset.url = urler(asset.name);

for (const u of asset.lineage.upstreams) {
    if (u.service !== 'kafka') continue;

    u.urn = u.urn.replace('.yonkou.io', '');
}
```

[`goja/goja_transform.go`](./goja/goja_transform.go)

#### Pros

- Ability to translate field names using the preferred struct tag. Won't work
  for protobuf tag out of the box but will be easy to support. However,
  our `json` and `protobuf` struct tag names are the same for now.
- Better error handling than [otto](#otto), uses error return value instead.
- Supports more modern JS constructs compared to [otto](#otto).
- Well maintained based on the 17 open issues and 8 pull requests.
- Although the library was evidently inspired by [otto](#otto) and was created
  after it became popular, goja is popular in its own right with 3.3K stars.

#### Cons

- No releases/tags for the library.
- No direct support for `context.Context`, workaround is possible using an
  'interrupt' - [goja#interrupting][goja-interrupting].
- The type information for Data field works against us and cannot be directly
  modified. We need to assign a second global field after unmarshaling the field
  of type `*anypb.Any`.
- Is not able to detect and protect against non-existent fields (silently does
  nothing).

### Bloblang

Docs: https://www.benthos.dev/docs/guides/bloblang/about
Reference: https://pkg.go.dev/github.com/benthosdev/benthos/v4/public/bloblang

#### Sample Script

```
asset.labels.script_engine = "bloblang"

map entity_name {
    root = this
    root.entity_name = match this.name {
        "ongoing_placed_and_waiting_acceptance_orders" => "customer_orders",
        "ongoing_orders" => "customer_orders",
        "merchant_avg_dispatch_arrival_time_10m" => "merchant_driver",
        "ongoing_accepted_orders" => "merchant_orders",
    }
}
asset.data.features = asset.data.features.map_each(f -> f.apply("entity_name"))

asset.owners = asset.owners.or([]).append({ "name": "Big Mom", "email": "big.mom@wholecakeisland.com" })

asset.url = urler(asset.name)

map urn_replace {
    root = this
    root.urn = this.urn.replace_all(".yonkou.io", "")
}

asset.lineage.upstreams = asset.lineage.upstreams.map_each(u -> if u.service == "kafka" {
    u.apply("urn_replace")
} else {
    u
})
```

[`bloblang/bloblang_transform.go`](./bloblang/bloblang_transform.go)

#### Pros

- Well documented without caveats that come with running a different language
  with subset of the API.

#### Cons

- No direct support for Go structs.
  See [benthos#1317(comment)][benthos-issues-1317-comment].
- We would need to pull in the dependencies
  of `github.com/benthosdev/benthos/v4` just for using the bloblang package. The
  dependencies of benthos are large in number (
  see [play-script-engine/network/dependencies][play-script-engine-nw-deps]).
  This impacts installing dependencies but not compilation or execution speed.
  To get a rough idea, here are the number of lines that we changed in `go.mod`
  and `go.sum` files:
  ```
  $ git diff --numstat
  20     1    go.mod
  100    5    go.sum
  ```
- Bloblang has the basic expectation of "take x and generate y using it". We
  want to transform x. Possible to hide it to some extent but can still get
  awkward.
- **Not able to fulfill the requirement** of 'Add a label to each entity.
  Ex: `"catch_phrase": "..."'`.
- Having to specify a blobl function each time we want to modify an object in an
  array is unpleasant.

### go-lua

[GitHub - Shopify/go-lua: A Lua VM in Go][go-lua]

#### Sample Script

```lua
if asset.labels == nil then
    asset.labels = {}
end
asset.labels["script_engine"] = "gopherlua"

for _, e in ipairs(asset.data.entities) do
    if e.labels == nil then
        e.labels = {}
    end
    e.labels["catch_phrase"] = "Here's Johnny!"
end

for _, f in ipairs(asset.data.features) do
    if f.name == "ongoing_placed_and_waiting_acceptance_orders" or f.name == "ongoing_orders" then
        f.entity_name = "customer_orders"
    elseif f.name == "merchant_avg_dispatch_arrival_time_10m" then
        f.entity_name = "merchant_driver"
    elseif f.name == "ongoing_accepted_orders" then
        f.entity_name = "merchant_orders"
    end
end

if asset.owners == nil then
    -- asset.owners gets initialised as a map and decoding into *asset.Asset fails.
    -- asset.owners = {}
end
-- Inserting into the table fails with runtime error: invalid key to 'next'
-- table.insert(asset.owners, {name = "Big Mom", email = "big.mom@wholecakeisland.com"})

asset.url = urler(asset.name)

for _, u in ipairs(asset.lineage.upstreams) do
    if u.service == "kafka" then
        -- Fails inexplicably with "attempt to call a nil value" runtime error.
        -- u.urn = u.urn:gsub(".yonkou.io", "")
    end
end
```

[`golua/golua_transform.go`](./golua/golua_transform.go)

#### Pros

- Being used in Shopify's load generation tool.
- Can use native lua API such as ipairs for working with values passed into lua
  script from Go.

#### Cons

- No direct support for Go structs . Need a good amount of reflection to
  simplify passing in and modifying a Go value inside lua. So we are instead
  passing in a `map[string]interface{}` into the script using a helper library.
  The modifications to the map do not reflect in the Go environment and we are
  having to pull out the modified value from lua after script execution
  finishes.
- No releases/tags for the library.
- No direct support for context.Context, and it is not clear how script
  execution could be terminated on timeout/context cancellation.
- **Not able to fulfill the requirements:**
  - Set the owner as `{Name: Big Mom, Email: big.mom@wholecakeisland.com}`.
  - For each lineage upstream, if the service is Kafka, apply a string replace
    op on the URN - `{.yonkou.io => }`.
- Is slower than GopherLua according to go-lua itself.

### GopherLua

[yuin/gopher-lua: GopherLua: VM and compiler for Lua in Go][gopherlua]

#### Sample Script

```lua
if asset.labels == nil then
    asset.labels = {}
end
asset.labels["script_engine"] = "gopherlua"

for _, e in data.entities() do
    if e.labels == nil then
        e.labels = {}
    end
    e.labels["catch_phrase"] = "You Shall Not Pass!"
end

for _, f in data.features() do
    if f.name == "ongoing_placed_and_waiting_acceptance_orders" or f.name == "ongoing_orders" then
        f.entityName = "customer_orders"
    elseif f.name == "merchant_avg_dispatch_arrival_time_10m" then
        f.entityName = "merchant_driver"
    elseif f.name == "ongoing_accepted_orders" then
        f.entityName = "merchant_orders"
    end
end

if asset.owners == nil then
    asset.owners = {}
end
asset.owners = asset.owners + {Name = "Big Mom", Email = "big.mom@wholecakeisland.com"}

asset.url = urler(asset.name)

for _, u in asset.lineage.upstreams() do
    if u.service == "kafka" then
        u.urn = u.urn:gsub("\.yonkou\.io", "")
    end
end
```

[`gopherlua/gopherlua_transform.go`](./gopherlua/gopherlua_transform.go)

#### Pros

- Lua's spec is designed to be an embeddable language with a relatively compact
  API and less ambiguity.
- Mature ecosystem of helper libraries.
  See [gopher-lua#libraries-for-gopherlua][gopher-lua-libraries-for-gopherlua].
- Is able to retain Go type information and directly modify fields. Is also able
  to detect and protect against assignment to or modification of non-existent
  fields (returns an error).
- Straightforward support for `context.Context`.
- Very popular library with 5.1K stars.

#### Cons

- The `layeh.com/gopher-luar` library makes it easy for us to pass in Go types
  into the lua script. But this comes at a cost and for some operations we would
  need to use a poorly documented subset of Lua's API. ex: Magical syntax for
  looping, appending. See https://pkg.go.dev/layeh.com/gopher-luar#New.
- The type information for `Data` field works against us and cannot be directly
  modified. We need to assign a second global field after unmarshaling the field
  of type `*anypb.Any`. Furthermore, we access fields by the Go field names
  where proto field names would have been more appropriate. Oddly, the fields
  can be accessed both as `asset.Urn` and `asset.urn`. Could be fixed with
  additional custom handling.
- No releases/tags for the library.

### Tengo

[GitHub - d5/tengo: A fast script language for Go][tengo] ([playground][tengo-playground])

#### Sample Script

[//]: # (@formatter:off)

```golang
text := import("text")

merge := func(m1, m2) {
    for k, v in m2 {
        m1[k] = v
    }
    return m1
}

asset.labels = merge({script_engine: "tengo"}, asset.labels)

for e in asset.data.entities {
    e.labels = merge({catch_phrase: "You talkin' to me?"}, e.labels)
}

for f in asset.data.features {
    if f.name == "ongoing_placed_and_waiting_acceptance_orders" || f.name == "ongoing_orders" {
        f.entity_name = "customer_orders"
    } else if f.name == "merchant_avg_dispatch_arrival_time_10m" {
        f.entity_name = "merchant_driver"
    } else if f.name == "ongoing_accepted_orders" {
        f.entity_name = "merchant_orders"
    }
}

asset.owners = append(asset.owners || [], { name: "Big Mom", email: "big.mom@wholecakeisland.com" })

asset.url = urler(asset.name)

for u in asset.lineage.upstreams {
    u.urn = u.service != "kafka" ? u.urn : text.replace(u.urn, ".yonkou.io", "", -1)
}
```

[//]: # (@formatter:on)

[`tengo/tengo_transform.go`](./tengo/tengo_transform.go)

#### Pros

- Straightforward support for `context.Context`.
- Option to implement interfaces defined by Tengo to be able to pass in
  user-defined structs. Cost and feasibility of implementation has not been
  evaluated.
- Pleasant syntax, albeit subjective.
- For the syntax, what is documented is what you get without caveats that come
  with running a different language with subset of the API.
- Good performance as per benchmarks created by library author.
- Popular library with 2.9K stars.

#### Cons

- No support for directly passing in user-defined types and limited support for
  passing in `map[string]interface{}`. Need to transform
  even `map[string]string` to `map[string]interface{}`.
- Development seems to have slowed down based on the 49 open issues, 13 pull
  requests and the pulse insights for the repo.

### Anko

[GitHub - mattn/anko: Scriptable interpreter written in golang][anko]

#### Sample Script

```
strings = import("strings")

func merge(m1, m2) {
    for k, v in m2 {
        m1[k] = v
    }
    return m1
}

asset.Labels = merge({"script_engine": "anko"}, asset.Labels)

for e in data.Entities {
    e.Labels = merge({"catch_phrase": "Take your stinking paws off me, you damn dirty ape!"}, e.Labels)
}

for f in data.Features {
    if f.Name == "ongoing_placed_and_waiting_acceptance_orders" || f.Name == "ongoing_orders" {
        f.EntityName = "customer_orders"
    } else if f.Name == "merchant_avg_dispatch_arrival_time_10m" {
        f.EntityName = "merchant_driver"
    } else if f.Name == "ongoing_accepted_orders" {
        f.EntityName = "merchant_orders"
    }
}

o = make(Owner)
o.Name = "Big Mom"
o.Email = "big.mom@wholecakeisland.com"
asset.Owners += o

asset.Url = urler(asset.Name)

for u in asset.Lineage.Upstreams {
    if u.Urn != "kafka" {
        continue
    }
    u.Urn = strings.Replace(u.Urn, ".yonkou.io", "", -1)
}
```

[`anko/anko_transform.go`](./anko/anko_transform.go)

#### Pros

- Straightforward support for `context.Context`.
- Documentation with examples.
- Can pass in Go structs directly and modify inside the script.
- For the syntax, what is documented is what you get without caveats that come
  with running a different language with subset of the API.

#### Cons

- A lot of reflection under the hood. Consequently would expect poor
  performance, probably the worst performance of all the options.
- The type information for Data field works against us and cannot be directly
  modified. We need to assign a second global field after unmarshaling the field
  of type `*anypb.Any`. Furthermore, we access fields by the Go field names
  where proto field names would have been more appropriate.
- Lot of insecure packages added by default and it is not clear how we could
  make specific packages unavailable. See [anko#327][anko-issues-327].
- Appending to slice of structs is clunky.
- Development and activity has slowed down on the repo with the last commit
  being nearly a year ago and no new issues or PRs created in the last month.

### gojq

[GitHub - itchyny/gojq: Pure Go implementation of jq][gojq]

#### Sample Script

```
.labels.script_engine = "gojq" |

.data.entities[].labels.catch_phrase = "Go ahead. Make my day." |

.data.features[] |=
    if .name == "ongoing_placed_and_waiting_acceptance_orders" or .name == "ongoing_orders" then
        .entity_name = "customer_orders"
    elif .name == "merchant_avg_dispatch_arrival_time_10m" then
        .entity_name = "merchant_driver"
    elif .name == "ongoing_accepted_orders" then
        .entity_name = "merchant_orders"
    else . end |

.owners += [{name: "Big Mom", email: "big.mom@wholecakeisland.com"}] |

.url = urler(.name) |

.lineage.upstreams[] |=
    if .service == "kafka" then .urn = (.urn | sub("\\.yonkou\\.io"; ""))  
    else . end
```

[`gojq/gojq_transform.go`](./gojq/gojq_transform.go)

#### Pros

- A bit surprising that it is able to fulfill all of the requirements
- jq is quite popular and commonly used for JSON transformations.
- Straightforward support for `context.Context`.
- A popular library with 2.3K stars considering it is not a general purpose
  scripting language.
- Fixes a bunch of issues in the original implementation of jq.

#### Cons

- Writing the script is hard to put together, unconventional.
- Some concepts such as defining a function inside the script probably won't be
  possible.
- No support for passing in user-defined types and limited support for passing
  in `map[string]interface{}`. Need to transform even `map[string]string` to
  `map[string]interface{}`.

[odpf-meteor-issues-420]: https://github.com/odpf/meteor/issues/420

[odpf-proton-asset-proto]: https://github.com/odpf/proton/blob/449cade/odpf/assets/v1beta2/asset.proto#L14

[otto]: https://github.com/robertkrimen/otto

[underscore-js]: https://underscorejs.org/

[otto-underscore]: http://github.com/robertkrimen/otto/tree/master/underscore

[otto-halting-problem]: https://github.com/robertkrimen/otto#halting-problem

[otto-caveat-emptor]: https://github.com/robertkrimen/otto#caveat-emptor

[goja]: https://github.com/dop251/goja

[goja-interrupting]: https://github.com/dop251/goja#interrupting

[benthos-issues-1317-comment]: https://github.com/benthosdev/benthos/issues/1317#issuecomment-1177361485

[play-script-engine-nw-deps]: https://github.com/sudo-suhas/play-script-engine/network/dependencies

[go-lua]: https://github.com/Shopify/go-lua

[gopherlua]: https://github.com/yuin/gopher-lua

[gopher-lua-libraries-for-gopherlua]: https://github.com/yuin/gopher-lua#libraries-for-gopherlua

[tengo]: https://github.com/d5/tengo

[tengo-playground]: https://tengolang.com/

[anko]: https://github.com/mattn/anko

[anko-issues-327]: https://github.com/mattn/anko/issues/327

[gojq]: https://github.com/itchyny/gojq
