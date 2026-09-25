BEGIN;

INSERT INTO advertisers (id,name,creation_idempotency_key,creation_request_fingerprint)
VALUES ('90000000-0000-4000-8000-000000000001','Phase 2 performance fixture','phase2-performance',decode(repeat('ab',32),'hex'));

-- Exact eligible cohorts: XB=1, XC=10, XD=100, XE=1000. XA has zero.
INSERT INTO campaigns (id,advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(10000 + n),12,'0'))::uuid,
       '90000000-0000-4000-8000-000000000001','Eligible ' || country_code || ' ' || n,
       'ACTIVE','search_results',1,'USD','eligible-' || country_code || '-' || n,decode(repeat('cd',32),'hex')
FROM (VALUES ('XB',1,0),('XC',10,1),('XD',100,11),('XE',1000,111)) AS cohorts(country_code,candidate_count,offset_value)
CROSS JOIN LATERAL generate_series(1 + offset_value,candidate_count + offset_value) AS n;

INSERT INTO campaign_target_countries (campaign_id,country_code)
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(10000 + n),12,'0'))::uuid,country_code
FROM (VALUES ('XB',1,0),('XC',10,1),('XD',100,11),('XE',1000,111)) AS cohorts(country_code,candidate_count,offset_value)
CROSS JOIN LATERAL generate_series(1 + offset_value,candidate_count + offset_value) AS n;

-- Rejected background: 25 DRAFT, 25 PAUSED, 25 wrong-placement, 25 wrong-country.
INSERT INTO campaigns (id,advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20000 + n),12,'0'))::uuid,
       '90000000-0000-4000-8000-000000000001'::uuid,'Background draft ' || n,
       'DRAFT','search_results',1,'USD','background-draft-' || n,decode(repeat('de',32),'hex')
FROM generate_series(1,25) AS n
UNION ALL
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20100 + n),12,'0'))::uuid,
       '90000000-0000-4000-8000-000000000001'::uuid,'Background paused ' || n,
       'PAUSED','search_results',1,'USD','background-paused-' || n,decode(repeat('de',32),'hex')
FROM generate_series(1,25) AS n
UNION ALL
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20200 + n),12,'0'))::uuid,
       '90000000-0000-4000-8000-000000000001'::uuid,'Background placement ' || n,
       'ACTIVE','home_feed',1,'USD','background-placement-' || n,decode(repeat('de',32),'hex')
FROM generate_series(1,25) AS n
UNION ALL
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20300 + n),12,'0'))::uuid,
       '90000000-0000-4000-8000-000000000001'::uuid,'Background country ' || n,
       'ACTIVE','search_results',1,'USD','background-country-' || n,decode(repeat('de',32),'hex')
FROM generate_series(1,25) AS n;

INSERT INTO campaign_target_countries (campaign_id,country_code)
SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20000 + n),12,'0'))::uuid,'XE' FROM generate_series(1,25) AS n
UNION ALL SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20100 + n),12,'0'))::uuid,'XE' FROM generate_series(1,25) AS n
UNION ALL SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20200 + n),12,'0'))::uuid,'XE' FROM generate_series(1,25) AS n
UNION ALL SELECT ('90000000-0000-4000-8000-' || lpad(to_hex(20300 + n),12,'0'))::uuid,'XF' FROM generate_series(1,25) AS n;

COMMIT;
