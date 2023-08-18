---

sidebar_position: 5
description: Common operators accross dialects

---

# Operators

Operators are methods on the dialect's `Expression`. There are a number of common operators shared by all supported dialects:

* `IsNull()`: X IS NULL
* `IsNotNull()`: X IS NOT NULL
* `IsDistinctFrom(y any)`: X IS DISTINCT FROM Y
* `IsNotDistinctFrom(y any)`: X IS NOT DISTINCT FROM Y
* `Minus(y any)`: X - Y
* `EQ(y any)`: X = Y
* `NE(y any)`: X \<\> Y
* `LT(y any)`: X \< Y
* `LTE(y any)`: X \<= Y
* `GT(y any)`: X \> Y
* `GTE(y any)`: X >= Y
* `In(...any)`: X IN (y, z)
* `NotIn(...any)`: X NOT IN (y, z)
* `Or(y any)`: X OR Y
* `And(y any)`: X AND Y
* `Concat(y any)`: X || Y
* `Between(y, z any)`: X BETWEEN Y AND Z
* `NotBetween(y, z any)`: X NOT BETWEEN Y AND Z

The following expressions cannot be chained and are expected to be used at the end of a chain

* `As(alias string)`: X as "alias". Used for aliasing column names

See dialect documentation for extra operators
