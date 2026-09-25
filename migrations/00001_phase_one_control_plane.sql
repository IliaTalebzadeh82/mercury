-- +goose Up
CREATE TABLE advertisers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    creation_idempotency_key text NOT NULL,
    creation_request_fingerprint bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT advertisers_name_valid CHECK (
        name = btrim(name) AND char_length(name) BETWEEN 1 AND 200
    ),
    CONSTRAINT advertisers_idempotency_key_valid CHECK (
        octet_length(creation_idempotency_key) BETWEEN 1 AND 128
    ),
    CONSTRAINT advertisers_fingerprint_valid CHECK (
        octet_length(creation_request_fingerprint) = 32
    ),
    CONSTRAINT advertisers_creation_key_unique UNIQUE (creation_idempotency_key)
);

CREATE TABLE placements (
    code text PRIMARY KEY,
    display_name text NOT NULL,
    CONSTRAINT placements_code_valid CHECK (code ~ '^[a-z][a-z0-9_]{0,62}$'),
    CONSTRAINT placements_display_name_valid CHECK (
        display_name = btrim(display_name) AND char_length(display_name) BETWEEN 1 AND 100
    )
);

INSERT INTO placements (code, display_name) VALUES
    ('home_feed', 'Home feed'),
    ('restaurant_list', 'Restaurant list'),
    ('search_results', 'Search results');

CREATE TABLE campaigns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    advertiser_id uuid NOT NULL,
    name text NOT NULL,
    state text NOT NULL DEFAULT 'DRAFT',
    placement_code text NOT NULL,
    budget_amount_minor bigint NOT NULL,
    currency text NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    creation_idempotency_key text NOT NULL,
    creation_request_fingerprint bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT campaigns_advertiser_fk FOREIGN KEY (advertiser_id)
        REFERENCES advertisers(id) ON DELETE RESTRICT,
    CONSTRAINT campaigns_placement_fk FOREIGN KEY (placement_code)
        REFERENCES placements(code) ON DELETE RESTRICT,
    CONSTRAINT campaigns_name_valid CHECK (
        name = btrim(name) AND char_length(name) BETWEEN 1 AND 200
    ),
    CONSTRAINT campaigns_state_valid CHECK (state IN ('DRAFT', 'ACTIVE', 'PAUSED', 'ENDED')),
    CONSTRAINT campaigns_budget_positive CHECK (budget_amount_minor > 0),
    CONSTRAINT campaigns_currency_supported CHECK (currency IN ('EUR', 'GBP', 'USD')),
    CONSTRAINT campaigns_version_positive CHECK (version >= 1),
    CONSTRAINT campaigns_idempotency_key_valid CHECK (
        octet_length(creation_idempotency_key) BETWEEN 1 AND 128
    ),
    CONSTRAINT campaigns_fingerprint_valid CHECK (
        octet_length(creation_request_fingerprint) = 32
    ),
    CONSTRAINT campaigns_creation_key_unique UNIQUE (advertiser_id, creation_idempotency_key)
);

CREATE INDEX campaigns_advertiser_created_idx
    ON campaigns (advertiser_id, created_at DESC, id DESC);

CREATE TABLE campaign_target_countries (
    campaign_id uuid NOT NULL,
    country_code text NOT NULL,
    CONSTRAINT campaign_target_countries_campaign_fk FOREIGN KEY (campaign_id)
        REFERENCES campaigns(id) ON DELETE CASCADE,
    CONSTRAINT campaign_target_countries_code_valid CHECK (country_code ~ '^[A-Z]{2}$'),
    PRIMARY KEY (campaign_id, country_code)
);

-- Serialize removals for the same campaign so two concurrent transactions
-- cannot each observe the country row being removed by the other transaction.
-- +goose StatementBegin
CREATE FUNCTION lock_campaign_for_country_removal() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM 1 FROM campaigns WHERE id = OLD.campaign_id FOR UPDATE;
        RETURN OLD;
    END IF;

    PERFORM 1 FROM campaigns
    WHERE id IN (OLD.campaign_id, NEW.campaign_id)
    ORDER BY id
    FOR UPDATE;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER target_removal_locks_campaign
    BEFORE DELETE OR UPDATE OF campaign_id ON campaign_target_countries
    FOR EACH ROW EXECUTE FUNCTION lock_campaign_for_country_removal();

-- A campaign and its targets are one aggregate. Deferring both checks allows
-- insert-campaign/insert-targets and delete-old/insert-new replacement flows,
-- while rejecting either a targetless insert or removal of the final target
-- when the transaction attempts to commit.
-- +goose StatementBegin
CREATE FUNCTION enforce_campaign_country_target() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    checked_campaign_id uuid;
BEGIN
    IF TG_TABLE_NAME = 'campaigns' THEN
        checked_campaign_id := NEW.id;
    ELSE
        checked_campaign_id := OLD.campaign_id;
    END IF;

    IF EXISTS (SELECT 1 FROM campaigns WHERE id = checked_campaign_id)
       AND NOT EXISTS (
           SELECT 1 FROM campaign_target_countries WHERE campaign_id = checked_campaign_id
       ) THEN
        RAISE EXCEPTION 'campaign must have at least one country target'
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER campaigns_require_country_target
    AFTER INSERT ON campaigns
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION enforce_campaign_country_target();

CREATE CONSTRAINT TRIGGER target_removal_preserves_campaign_country
    AFTER DELETE OR UPDATE OF campaign_id ON campaign_target_countries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION enforce_campaign_country_target();

-- +goose Down
DROP TABLE campaign_target_countries;
DROP TABLE campaigns;
DROP FUNCTION enforce_campaign_country_target();
DROP FUNCTION lock_campaign_for_country_removal();
DROP TABLE placements;
DROP TABLE advertisers;
