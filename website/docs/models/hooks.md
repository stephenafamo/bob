---

sidebar_position: 5
description: Hook into model operations

---

# Hooks

Hooks are a way to perform actions before/after a model operation.

View Models have:

* `AfterSelectHooks`

In **addition**, TableModels have:

* `BeforeInsertHooks`
* `AfterInsertHooks`
* `BeforeUpsertHooks`
* `AfterUpsertHooks`
* `BeforeUpdateHooks`
* `AfterUpdateHooks`
* `BeforeDeleteHooks`
* `AfterDeleteHooks`

These hooks run at the point one would expect from their naming.

## Writing a Hook

A hook has the signature:

```go
func myHook(ctx context.Context, exec bob.Executor, t T) (context.Context, error) {
    return ctx, nil
}
```

The returned context is passed to the next registered hook and finally to the query.

## Registering hooks

A hook can be registered with the `Add` method:

```go
userTable.BeforeUpdateHooks.Add(myHook)
```

## Skipping hooks

If you need to run a query without hooks, use the `SkipHooks` function:

```go
// Hooks are skipped
userTable.Select(bob.SkipHooks(ctx), exec).All()
```
