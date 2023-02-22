---

sidebar_position: 20
description: Features ignored by Bob

---

# Ignored features

## Automatic timestamps (createdAt/UpdatedAt)

While convenient to have this in the ORM, it is much better to implement this at the DB level. Therefore, there are no plans to implement this in Bob. It isn't worth the additional complexity.

## Soft deletes

There are no immediate plans for this. The many edge cases make this extremely complex especially when relationships and cascading soft deletes are considered.
