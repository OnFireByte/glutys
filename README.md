# Glutys

Glutys (Glue to ts) is [tRPC](https://trpc.io/) inspired typesafe TypeScript Front-End to Go Back-End by generating "contract" from registered function.

For client source code: [Glutys-client repository](https://github.com/OnFireByte/glutys-client)

## Feature

-   Typesafe and autocompletion on TypeScript
-   Back-End's based on net/http package
-   Front-End's based on axios
-   "Context" abstraction to get data from request in RPC function.

**Glutys doesn't have proper docs yet, so you should check the [example project](https://github.com/OnFireByte?tab=repositories) while reading the README**

## Installation

You can get the package by go get command

```bash
go get github.com/OnFireByte/glutys
```

## Setting up

Glutys require you to create a seperate main function that act as script to generating code. I recommend following [Go project layout](https://github.com/golang-standards/project-layout/tree/master/cmd) by creating `cmd/glutys/main.go` for generate script, and `cmd/main/main.go` for main application.

I also recommend the monorepo structure since it's easier to manage the generated contract.

```go
// cmd/glutys/main.go
package main

import (
    "os"
    "server/route"

    "github.com/onfirebyte/glutys"
)

func main() {
    fmt.Println("Generating routes...")

    builder := glutys.NewBuilder("server/generated/routegen")
    builder.AddContextParser(reqcontext.ParseUsername)

    builder.CreateRouter(route.RootRoute)

    goFileString, tsFileString := builder.Build()

    file, err := os.Create("generated/routegen/route.go")
    if err != nil {
    panic(err)
    }

    file.WriteString(goFileString)

    tsFile, err := os.Create("../client/generated/contract.ts")
    if err != nil {
    panic(err)
    }

    tsFile.WriteString(tsFileString)

    fmt.Println("Done!")
}
```

```go
// cmd/main/main.go
package main

import (
    "fmt"
    "net/http"
    "server/generated/routegen"
)

func main() {
    handler := routegen.NewHandler()
    http.HandleFunc("/api", handler.Handle)

    fmt.Println("Listening on port 8080")
    http.ListenAndServe(":8080", nil)
}
```

## Creating new RPC

1. Creating function anywhere in your project **except the glutys/main.go file**

```go
// route/math/math.go
package math

func Fib(n int) int {
   if n <= 1 {
   return n
   }

   a := 0
   b := 1
   for i := 2; i <= n; i++ {
   a, b = b, a+b
   }

   return b
}
```

2. Adding new route into `builder.CreateRouter` in generate script

```go
// cmd/glutys/main.go
func main() {
    builder := glutys.NewBuilder("server/generated/routegen")

    ...

    builder.CreateRouter(map[string][]any{
    "math.fib":        {math.Fib},
    })

    ...

    goFileString, tsFileString := builder.Build()
}
```

now you can call it from client!

```typescript
import { CreateAPIClient } from "glutys-client";
import { GlutysContract } from "./generated/contract";

const instance = axios.create({
    baseURL: "http://localhost:8080/api",
    headers: {
        "user-token": "1234",
    },
});

const api = CreateAPIClient<GlutysContract>(instance);

console.log(api.math.fib(5));
```

Note:

1. Glutys also support multiple argument, struct as argument and return data.
2. RPC function in go can return error as second value, the response will be 400 with json message if the error is not nil.

## Creating context parser

"context" in this context (no pun intended) is the data that you need to process in RPC function that doesn't come as argument, for example, the user token that attached with request header, you can ceate function that parsing these data and pass it into RPC function

1. Create parsing function

```go
package contextval

import (
    "fmt"
    "net/http"
)

// uniquee type for context is required since
// glutys uses type name to map the context
type UserContext string

func GetUserContext(r *http.Request) (UserContext, error) {
    // get user token from header
    userID := r.Header.Get("user-token")
    if userID == "" {
    return "", fmt.Errorf("userToken header not found")
    }
    return UserContext(userID), nil
}
```

2. Add the parsing function to generate script

```go
// cmd/glutys/main.go
func main() {

    ...

    builder.AddContextParser(contextval.GetUserContext)

    ...

    goFileString, tsFileString := builder.Build()

    ...
}
```

3. Now you can use context in your RPC function

```go
func SayHello(userToken contextval.UserContext, name string) string {
    return fmt.Sprintf("Hello %v!, your token is %v", name, userToken)
}
```

```ts
api.sayHello("John"); // Hello John, your token is 1234.
```

## Adding dependencies

Similar to context, if you want to do dependencies injection, you can pass the dependencies as argument. The dependency can be both real type or interface.

For example, we have dependency `cache.Cache`.

```go
type Cache interface {
    Get(key string) (string, bool)
    Set(key string, value string)
}

type CacheImpl struct {
    cache map[string]string
}

func NewCacheImpl() *CacheImpl {
    return &CacheImpl{cache: map[string]string{}}
}

func (c *CacheImpl) Get(key string) (string, bool) {
    v, ok := c.cache[key]
    return v, ok
}

func (c *CacheImpl) Set(key string, value string) {
    c.cache[key] = value
}
```

1. Add the dependency type to building script

```go
// cmd/glutys/main.go
func main() {

    ...

    // You must use pointer to type, not the type itself
    builder.AddDependencyType((*cache.Cache)(nil))

    ...

    goFileString, tsFileString := builder.Build()
}
```

2. Generate the code. Then add the dependency in `NewHanlder` function.

```go
// cmd/main/main.go
func main() {
    // the order of dependencies depends on the order of AddDependencyType calls
    handler := routegen.NewHandler(
    cache.NewCacheImpl(),
    )
    http.HandleFunc("/api", handler.Handle)

    ...
}
```

3. Now you can use it in RPC function

```go
func Fib(cache cache.Cache, n int) (int, error) {
	if raw, ok := cache.Get(strconv.Itoa(n)); ok {
		return strconv.Atoi(raw)
	}

	...

	cache.Set(strconv.Itoa(n), strconv.Itoa(result))

	return result, nil
}
```

## Adding custom type

When you add a type that doesn't supported by JSON specification, you can use `builder.AddCustomType` to tell glutys to map the custom type to proper TS type. Note that you have marshalling process that correctly convert it to match the TS type that you specified

For example, if you want to use UUID from `github.com/google/uuid`

```go
// cmd/glutys/main.go
func main() {
    ...

    // uuid.UUID already have marshall method that convert to string.
    // arg: value of that type, matched TS type
    builder.AddCustomType(uuid.UUID{}, "string")

    ...

    goFileString, tsFileString := builder.Build()
}
```

```go
// RPC function
func GetUUIDBase64(id uuid.UUID) string {
    return base64.StdEncoding.EncodeToString(id[:])
}
```

```ts
// Client
console.log(await api.GetUUIDBase64("123e4567-e89b-12d3-a456-426655440000")); //Ej5FZ+ibEtOkVkJmVUQAAA==
```

## Todo Feature

-   [ ] Route specific middleware
-   [x] Axios client option

## Limitaion

-   Can't declare anonymous function in generated file (both route handler and context parser).
