
WITH
"_sl_src" AS MATERIALIZED (SELECT 
q.*, row_number() OVER () AS "_sl_ord"
FROM (SELECT id FROM tk_root) AS "q"
)
SELECT 
0 AS _sl_set, "_sl_src"."_sl_ord", "_sl_src"."id", NULL::int4 AS "id", NULL::int4 AS "root_id", NULL::int4 AS "other_key", NULL::int4 AS "id", NULL::int4 AS "root_id", NULL::int4 AS "other_key"
FROM "_sl_src"
UNION ALL (SELECT 
1 AS "_sl_set", (CAST(NULL AS bigint)) AS "_sl_ord", (CAST(NULL AS int4)) AS "id", "_sl_t"."id", "_sl_t"."root_id", "_sl_t"."other_key", (CAST(NULL AS int4)) AS "id", (CAST(NULL AS int4)) AS "root_id", (CAST(NULL AS int4)) AS "other_key"
FROM "tk_mid" AS "_sl_t"
WHERE (("_sl_t"."root_id") IN ((SELECT 
"id"
FROM "_sl_src"
)))
)
UNION ALL (SELECT 
2 AS "_sl_set", (CAST(NULL AS bigint)) AS "_sl_ord", (CAST(NULL AS int4)) AS "id", (CAST(NULL AS int4)) AS "id", (CAST(NULL AS int4)) AS "root_id", (CAST(NULL AS int4)) AS "other_key", "_sl_t"."id", "_sl_t"."root_id", "_sl_t"."other_key"
FROM "tk_mid" AS "_sl_t"
WHERE (("_sl_t"."root_id") IN ((SELECT 
"id"
FROM "_sl_src"
)))
)
