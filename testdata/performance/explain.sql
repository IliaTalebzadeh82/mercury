\set opportunity_id 'a0000000-0000-4000-8000-000000000001'
\set placement 'search_results'

\echo 'COHORT XA: 0 eligible'
\set country 'XA'
EXPLAIN (ANALYZE, BUFFERS)
WITH eligible AS (
    SELECT c.id,c.version,decode(md5(:'opportunity_id'::uuid::text || ':' || c.id::text),'hex') AS ranking_score
    FROM campaign_target_countries AS t JOIN campaigns AS c ON c.id=t.campaign_id
    WHERE t.country_code=:'country' AND c.state='ACTIVE' AND c.placement_code=:'placement'
), ranked AS (
    SELECT id,version,count(*) OVER () AS eligible_candidate_count FROM eligible
    ORDER BY ranking_score DESC,id ASC LIMIT 1
)
SELECT p.code,ranked.id::text,ranked.version,COALESCE(ranked.eligible_candidate_count,0)
FROM placements AS p LEFT JOIN ranked ON true WHERE p.code=:'placement';

\echo 'COHORT XB: 1 eligible'
\set country 'XB'
EXPLAIN (ANALYZE, BUFFERS)
WITH eligible AS (
    SELECT c.id,c.version,decode(md5(:'opportunity_id'::uuid::text || ':' || c.id::text),'hex') AS ranking_score
    FROM campaign_target_countries AS t JOIN campaigns AS c ON c.id=t.campaign_id
    WHERE t.country_code=:'country' AND c.state='ACTIVE' AND c.placement_code=:'placement'
), ranked AS (
    SELECT id,version,count(*) OVER () AS eligible_candidate_count FROM eligible
    ORDER BY ranking_score DESC,id ASC LIMIT 1
)
SELECT p.code,ranked.id::text,ranked.version,COALESCE(ranked.eligible_candidate_count,0)
FROM placements AS p LEFT JOIN ranked ON true WHERE p.code=:'placement';

\echo 'COHORT XC: 10 eligible'
\set country 'XC'
EXPLAIN (ANALYZE, BUFFERS)
WITH eligible AS (
    SELECT c.id,c.version,decode(md5(:'opportunity_id'::uuid::text || ':' || c.id::text),'hex') AS ranking_score
    FROM campaign_target_countries AS t JOIN campaigns AS c ON c.id=t.campaign_id
    WHERE t.country_code=:'country' AND c.state='ACTIVE' AND c.placement_code=:'placement'
), ranked AS (
    SELECT id,version,count(*) OVER () AS eligible_candidate_count FROM eligible
    ORDER BY ranking_score DESC,id ASC LIMIT 1
)
SELECT p.code,ranked.id::text,ranked.version,COALESCE(ranked.eligible_candidate_count,0)
FROM placements AS p LEFT JOIN ranked ON true WHERE p.code=:'placement';

\echo 'COHORT XD: 100 eligible'
\set country 'XD'
EXPLAIN (ANALYZE, BUFFERS)
WITH eligible AS (
    SELECT c.id,c.version,decode(md5(:'opportunity_id'::uuid::text || ':' || c.id::text),'hex') AS ranking_score
    FROM campaign_target_countries AS t JOIN campaigns AS c ON c.id=t.campaign_id
    WHERE t.country_code=:'country' AND c.state='ACTIVE' AND c.placement_code=:'placement'
), ranked AS (
    SELECT id,version,count(*) OVER () AS eligible_candidate_count FROM eligible
    ORDER BY ranking_score DESC,id ASC LIMIT 1
)
SELECT p.code,ranked.id::text,ranked.version,COALESCE(ranked.eligible_candidate_count,0)
FROM placements AS p LEFT JOIN ranked ON true WHERE p.code=:'placement';

\echo 'COHORT XE: 1000 eligible'
\set country 'XE'
EXPLAIN (ANALYZE, BUFFERS)
WITH eligible AS (
    SELECT c.id,c.version,decode(md5(:'opportunity_id'::uuid::text || ':' || c.id::text),'hex') AS ranking_score
    FROM campaign_target_countries AS t JOIN campaigns AS c ON c.id=t.campaign_id
    WHERE t.country_code=:'country' AND c.state='ACTIVE' AND c.placement_code=:'placement'
), ranked AS (
    SELECT id,version,count(*) OVER () AS eligible_candidate_count FROM eligible
    ORDER BY ranking_score DESC,id ASC LIMIT 1
)
SELECT p.code,ranked.id::text,ranked.version,COALESCE(ranked.eligible_candidate_count,0)
FROM placements AS p LEFT JOIN ranked ON true WHERE p.code=:'placement';
