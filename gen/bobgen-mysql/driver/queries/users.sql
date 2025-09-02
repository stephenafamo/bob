-- SelectUsers
SELECT * FROM `users` WHERE id IN (?)
;

-- SelectUsersCount
SELECT count(id) /*:int*/ as users_count
FROM `users`
WHERE id IN (?)
;

-- SelectUsersUnion
SELECT id FROM `users` WHERE id IN (?)
UNION SELECT id FROM `users` WHERE id IN (?)
;

-- SelectUsersUnionInParens
SELECT id FROM `users` WHERE id IN (?)
UNION (SELECT id FROM `users` WHERE id IN (?))
;

-- GetQueryIDByID
SELECT query.id
FROM query
WHERE query.id = ?
;

