---

sidebar_position: 1

---

# Bob vs GORM

## Source of truth

* **GORM** is code-first. It generates the database from your code, using GORM-specific techniques. If you use a different tool for migration, you then have to manually ensure that your GORM models are in sync with your database.

* **Bob** is database-first. You generate your database using the first class tools for your DB (SQL, pgAdmin, migration pacakges). And then generate your models from your database.

## Type Safety

* **GORM** makes heavy use of `interface{}` and magic strings which will lead to runtime panics if incorrect.

* **Bob** generates type-safe code for every thing, including loading relationships. Any incorrect use will be caught compile time and not a runtime.

## Query Building

* **GORM** is limited to a subset of common features across the supported dialects.

* **Bob** has a much more powerful query builder. Across all supported dialects, Bob allows you to build pretty much any Select, Update, Insert or Delete query supported by the dialect.

## Ecosystem

* **GORM** needs special drivers to work. However because of its popularity, there are existing plugins for many use cases.

* **Bob** works with `*sql.DB`, so any package that works with the standard library will work with Bob.

## Adoption curve

* **GORM** needs to be adopted fully. It is not possible to use only a part of its features.

* **Bob** can be used incrementally. For example, it is possible to use only the query builder without needing to fully adopt Bob.

## Testability

* **GORM** does not provide any testing helpers.

* **Bob** also generates factories (inspired by Ruby's FactoryBot) to make it very easy to test your models.

