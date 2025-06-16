---
sidebar_position: 4
description: Execute a query and return a slice of objects.
---

# All

Execute a query and return a slice of type representing the results of the query
Similar to `QueryContext`, but works directly on a `bob.Query` object.

This function is a wrapper around [`scan.All`](https://pkg.go.dev/github.com/stephenafamo/scan#All).

```go
type userObj struct {
    ID int
    Name string
}

ctx := context.Background()
db, err := bob.Open("postgres", "...")
if err != nil {
    // ...
}

q := psql.Select(...)

// user is of type []userObj{}
users, err := bob.All(ctx, db, q, scan.StructMapper[userObj]())
if err != nil {
    // ...
}
```

There is also the `Allx` function. This version of the `All` function takes a type parameter for a Transformer that will be used to transform the result into a different type.

For common use cases, you can use the `bob.SliceTransformer` to cast the returned slice to a concrete slice type.

```go
type userSlice []userObj

func (u userSlice) MethodOnSliceType() {}

// users is of type userSlice
users, err := bob.Allx[bob.SliceTransformer[userObj, userSlice]](ctx, db, q, scan.StructMapper[userObj]())
if err != nil {
    // ...
}
```
