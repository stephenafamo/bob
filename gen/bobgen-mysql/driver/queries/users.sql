-- SelectUsers
SELECT * FROM `users` WHERE id IN (?)
;

-- SelectUsersCount
SELECT count(id) /*:int*/ as users_count
FROM `users`
WHERE id IN (?)
;
