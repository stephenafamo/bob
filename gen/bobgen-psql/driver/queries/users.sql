-- SelectUsers
SELECT * FROM users WHERE id IN ($1)
;

-- SelectUsersTableSample
SELECT u.* FROM users AS u TABLESAMPLE BERNOULLI(50) REPEATABLE (7)
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


-- MergeUser
MERGE INTO users AS target
USING (VALUES ($1::int, $2::varchar)) AS source (id, primary_email)
ON target.id = source.id
WHEN MATCHED THEN UPDATE SET primary_email = source.primary_email
WHEN NOT MATCHED THEN INSERT (id, primary_email) VALUES (source.id, source.primary_email)
;


-- SearchUsersByTerms
WITH input_terms AS (
    SELECT DISTINCT term
    FROM unnest($1::text[]) AS term
    WHERE term <> ''
)
SELECT primary_email
FROM (
    SELECT
        u.id,
        u.primary_email,
        count(*) FILTER (
            WHERE u.primary_email ILIKE ('%' || input_terms.term || '%')
        ) AS matched_term_count
    FROM users u
    LEFT JOIN input_terms ON TRUE
    GROUP BY u.id, u.primary_email
) ranked
ORDER BY matched_term_count DESC, id ASC
LIMIT 5
;
