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

There is also the `Allx` function. The main difference is that it takes 2 type parameters instead of one.

The 2nd type parameter indicates the type of the slice to be returned. This is useful if you have methods defined on the slice type and do not want to always do the type cast yourself.


```go
type userSlice []userObj

func (u userSlice) MethodOnSliceType() {}

// users is of type userSlice
users, err := bob.Allx[userObj, userSlice](ctx, db, q, scan.StructMapper[userObj]())
if err != nil {
    // ...
}
```
