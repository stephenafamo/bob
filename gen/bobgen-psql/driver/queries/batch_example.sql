-- InsertUserBatch :::batch
INSERT INTO users (id, primary_email) VALUES ($1, $2)
RETURNING *
;

-- SelectUsersBatch :::batch
SELECT * FROM users WHERE id = $1
;

-- UpdateUserBatch :::batch
UPDATE users SET primary_email = $1 WHERE id = $2
RETURNING *
;

-- DeleteUserBatch :::batch
DELETE FROM users WHERE id = $1
;
