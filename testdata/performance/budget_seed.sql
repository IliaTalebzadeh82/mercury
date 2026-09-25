BEGIN;

INSERT INTO advertisers (id,name,creation_idempotency_key,creation_request_fingerprint)
VALUES ('91000000-0000-4000-8000-000000000000','Phase 3 performance fixture','phase3-performance',decode(repeat('91',32),'hex'));

INSERT INTO campaigns
    (id,advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
VALUES
    ('91000000-0000-4000-8000-000000000001','91000000-0000-4000-8000-000000000000','Hot accounting','ACTIVE','home_feed',1000000000,'EUR','phase3-hot',decode(repeat('92',32),'hex')),
    ('91000000-0000-4000-8000-000000000002','91000000-0000-4000-8000-000000000000','Replay accounting','ACTIVE','home_feed',1000000000,'EUR','phase3-replay',decode(repeat('92',32),'hex')),
    ('91000000-0000-4000-8000-000000000003','91000000-0000-4000-8000-000000000000','Insufficient accounting','ACTIVE','home_feed',1,'EUR','phase3-insufficient',decode(repeat('92',32),'hex'));

INSERT INTO campaigns
    (id,advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
SELECT ('91000000-0000-4000-8000-' || lpad(to_hex(256+n),12,'0'))::uuid,
       '91000000-0000-4000-8000-000000000000','Distributed accounting ' || n,
       'ACTIVE','home_feed',1000000000,'EUR','phase3-spread-' || n,decode(repeat('93',32),'hex')
FROM generate_series(0,99) AS n;

INSERT INTO campaign_target_countries (campaign_id,country_code)
SELECT id,'DE' FROM campaigns WHERE advertiser_id='91000000-0000-4000-8000-000000000000';

UPDATE campaigns SET committed_spend_minor=1
WHERE id='91000000-0000-4000-8000-000000000003';

INSERT INTO budget_consumption_commands
    (campaign_id,idempotency_key,request_fingerprint,amount_minor,currency,outcome,
     campaign_state_at_evaluation,configured_budget_minor,committed_spend_before_minor,
     resulting_committed_spend_minor,resulting_remaining_budget_minor)
VALUES
    ('91000000-0000-4000-8000-000000000003','phase3-seed-spend',decode(repeat('94',32),'hex'),
     1,'EUR','APPROVED','ACTIVE',1,0,1,0);

COMMIT;
