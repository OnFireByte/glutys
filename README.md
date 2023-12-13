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

	builder := glutys.Builder{
		GeneratePath: "server/generated/routegen",
	}

    builder.AddContextParser(user.GetUserContext)

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
	http.HandleFunc("/api", routegen.RouteHandler)

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

	builder := glutys.Builder{
		GeneratePath: "server/generated/routegen",
	}

	builder.CreateRouter(map[string][]any{
	    "math.fib":        {math.Fib},
    })
    ...
}
```

now you can call it from client!

```typescript
import { CreateAPIClient } from "glutys-client";
import { GlutysContract } from "./generated/contract";

const api = CreateAPIClient<GlutysContract>("http://localhost:8080/api");

console.log(api.math.fib(5));
```

Note:

1. Glutys also support multiple argument, struct as argument and return data.
2. RPC function in go can return error as second value, the response will be 400 with json message if the error is not nil.

## Create context parser

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
	userID := r.Header.Get("user token")
	if userID == "" {
		return "", fmt.Errorf("userToken header not found")
	}
	return UserContext(userID), nil
}
```

2. add the parsing function to generate script

```go
// cmd/glutys/main.go
func main() {

	builder := glutys.Builder{
		GeneratePath: "server/generated/routegen",
	}

    builder.AddContextParser(contextval.GetUserContext)

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
api.sayHello("John"); // Hello John, your token is 0.
```

## Adding custom type

When you add a type that doesn't supported by JSON specification, you can use `builder.AddCustomType` to tell glutys to map the custom type to proper TS type. Note that you have marshalling process that correctly convert it to match the TS type that you specified

For example, if you want to use UUID from `github.com/google/uuid`

```go
// cmd/glutys/main.go
func main() {

	builder := glutys.Builder{
		GeneratePath: "server/generated/routegen",
	}

    // uuid.UUID already have marshall method that convert to string.
    // arg: value of that type, matched TS type
    builder.AddCustomType(uuid.UUID{}, "string")

    ...
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

-   [] Route specific middleware
-   [] Axios client option

## Limitaion

-   Can't declare anonymous function in generated file (both route handler and context parser).
