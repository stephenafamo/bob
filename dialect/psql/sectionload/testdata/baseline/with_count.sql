
WITH
"_sl_src" AS MATERIALIZED (SELECT 
q.*, row_number() OVER () AS "_sl_ord"
FROM (SELECT id FROM count_parents) AS "q"
)
SELECT 
0 AS _sl_set, "_sl_src"."_sl_ord", "_sl_src"."id", NULL::int4 AS "id", NULL::int4 AS "parent_id", NULL::int4 AS "parent_id", NULL::bigint AS "_sl_cnt2"
FROM "_sl_src"
UNION ALL (SELECT 
1 AS "_sl_set", (CAST(NULL AS bigint)) AS "_sl_ord", (CAST(NULL AS int4)) AS "id", "_sl_t"."id", "_sl_t"."parent_id", (CAST(NULL AS int4)) AS "parent_id", (CAST(NULL AS bigint)) AS "_sl_cnt2"
FROM "row_children" AS "_sl_t"
WHERE (("_sl_t"."parent_id") IN ((SELECT 
"id"
FROM "_sl_src"
)))
)
UNION ALL (SELECT 
2 AS "_sl_set", (CAST(NULL AS bigint)) AS "_sl_ord", (CAST(NULL AS int4)) AS "id", (CAST(NULL AS int4)) AS "id", (CAST(NULL AS int4)) AS "parent_id", "_sl_t"."parent_id" AS "parent_id", "_sl_t"."_sl_cnt2" AS "_sl_cnt2"
FROM (SELECT 
"_sl_t"."parent_id", count(*) AS "_sl_cnt2"
FROM "count_children" AS "_sl_t"
WHERE (("_sl_t"."parent_id") IN ((SELECT 
"id"
FROM "_sl_src"
)))
GROUP BY "_sl_t"."parent_id"
) AS "_sl_t"
)
