# Delete

## Simple

SQL:

```sql
DELETE FROM films WHERE (`kind` = ?)
```

Args:

* `"Drama"`

Code:

```go
mysql.Delete(
  dm.From("films"),
  dm.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
)
```

## Multiple Tables

SQL:

```sql
DELETE FROM films, actors USING films
INNER JOIN film_actors ON ((films.id) = (film_actors.film_id))
INNER JOIN actors ON ((film_actors.actor_id) = (actors.id)) WHERE (`kind` = ?)
```

Args:

* `"Drama"`

Code:

```go
mysql.Delete(
  dm.From("films"),
  dm.From("actors"),
  dm.Using("films"),
  dm.InnerJoin("film_actors").OnEQ(mysql.Raw("films.id"), mysql.Raw("film_actors.film_id")),
  dm.InnerJoin("actors").OnEQ(mysql.Raw("film_actors.actor_id"), mysql.Raw("actors.id")),
  dm.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
)
```

## With Limit And Offest

SQL:

```sql
DELETE FROM films WHERE (`kind` = ?) ORDER BY producer DESC LIMIT 10
```

Args:

* `"Drama"`

Code:

```go
mysql.Delete(
  dm.From("films"),
  dm.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
  dm.Limit(10),
  dm.OrderBy("producer").Desc(),
)
```

## With Using

SQL:

```sql
DELETE FROM employees USING accounts WHERE (`accounts`.`name` = ?) AND (`employees`.`id` = `accounts`.`sales_person`)
```

Args:

* `"Acme Corporation"`

Code:

```go
mysql.Delete(
  dm.From("employees"),
  dm.Using("accounts"),
  dm.Where(mysql.Quote("accounts", "name").EQ(mysql.Arg("Acme Corporation"))),
  dm.Where(mysql.Quote("employees", "id").EQ(mysql.Quote("accounts", "sales_person"))),
)
```
