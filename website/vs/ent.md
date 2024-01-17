---

sidebar_position: 2

---

# Bob vs Ent

## Source of truth

* **Ent** is code-first. The schema is defined in code and it then generates the entities using Ent-specific techniques. If you use a different tool for migration, you then have to manually ensure that your Ent schema is in sync with your database.

* **Bob** is database-first. You generate your database using the first class tools for your DB (SQL, pgAdmin, migration packages). And then generate your models from your database.

## Query Building

* **Ent** provides predicates to build queries. This covers common usecases and allows for easy graph traversal.

* **Bob** has a more powerful query builder. Across all supported dialects, Bob allows you to build pretty much any Select, Update, Insert or Delete query supported by the dialect.

## Migrations

* **Ent** expects to handle your migrations. It includes automatic migrations for development/testing and versioned migrations powered by Atlas

* **Bob** does not deal with migrations at all. You can use any tool of your choice for migrations (golang-migrate, flyway, atlas, e.t.c) and then regenerate your models whenever the database state is updated.

## Ecosystem

* **Ent** currently has a more mature ecosystem. For example it includes plugins to integrate with gqlgen and gRPC.

* **Bob** is only focused on the data layer. Although it is possible to hook into the generation process to integrate with GraphQL or gRPC, it is not done at this time.

## Adoption curve

* **Ent** needs to be adopted fully. It is not possible to use only a part of its features.

* **Bob** can be used incrementally. For example, it is possible to use only the query builder without needing to fully adopt Bob.
