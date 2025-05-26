-- SelectUsers
SELECT * FROM users WHERE id IN ($1)
;

-- InsertUser
INSERT INTO users (id, primary_email) VALUES ($1, $2)
;


-- BulkInsertUsers
INSERT INTO users (id, primary_email) VALUES ($1, $2), ($3, $4)
;


-- InsertFromSelect
INSERT INTO users (id, primary_email)
SELECT id, primary_email FROM users WHERE id IN ($1)
;


-- UpdateUser
UPDATE ONLY users SET primary_email = $1 WHERE id = $2
RETURNING users.*
;

-- DeleteUser
DELETE FROM users WHERE id = $1;
