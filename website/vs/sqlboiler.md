---

sidebar_position: 3

---

# Bob vs SQLBoiler

Bob is primarily inspired by SQLBoiler so there are **a lot** of similarities between the two of them.
I actually started Bob as an experiment for how v5 of SQLBoiler could look, but it ended up looking quite different and I wouldn't want to force current SQLBoiler users into a tedious migration.

Bob's new foundation made it possible to do many things SQLBoiler cannot since Bob does not need to worry about backwards compatibility.

* Query Object:
  * **SQLBoiler**: The same query object is shared for all dialects and query types.
  * **Bob**: Each query type in each dialect is unique.
* Query Mods:
  QueryMods are how both packages build queries. For example, to add a `WHERE` clause.

  * **SQLBoiler**: The same query mods are used every where, making it possible to craft invalid queries where a mod is used on the wrong query type (e.g. JOINS on a DELETE query)
  * **Bob**: Each query type has its own mods, making it impossible to craft an invalid query.
* Dialect Support:
  * **SQLBoiler**: Since every feature has to work accross all dialects, it is difficult to support the full range of a dialect or add new dialects.
  * **Bob**: Since every query and its mods are independent of each other, Bob can support the full range of capabilities of any dialect, and add new dialects without being concerned about the existing ones.
* Building Custom SQL:
  * **SQLBoiler**: Outside of using the generated code, the user has to manage SQL building by hand which can involve a lot of manual string manipulation.
  * **Bob**: Bob uses expressions and provides lots of utilities to fluently build SQL queries without having to do any string manipulation.

## New Features in Bob

1. Cross schema generation. SQLBoiler currently does not support this, although technically it could.
1. Preloading with LEFT JOINs
1. Multi-Key relationships
1. Context chaining in hooks
1. Relationship across tables (has-one-through, has-many-through)
1. Generate easier-to-use expression builders. This is not easy to explain, but because of how the Query builder works, the chances of having to use non-typed strings are even less.

## SQLBoiler features that are not implemented in Bob

1. Automatic timestamps (createdAt/updatedAt): While convenient to have this in the ORM, it is much better to implement this at the DB level. There are no plans to implement this in Bob. It isn't worth the additional complexity.
1. Soft deletes: The many edge cases make this extremely complex especially when relationships and cascading soft deletes are considered.
