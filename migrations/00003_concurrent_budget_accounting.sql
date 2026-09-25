-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN committed_spend_minor bigint NOT NULL DEFAULT 0,
    ADD CONSTRAINT campaigns_committed_spend_nonnegative CHECK (committed_spend_minor >= 0),
    ADD CONSTRAINT campaigns_budget_covers_committed CHECK (budget_amount_minor >= committed_spend_minor);

CREATE TABLE budget_consumption_commands (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id uuid NOT NULL,
    idempotency_key text NOT NULL,
    request_fingerprint bytea NOT NULL,
    amount_minor bigint NOT NULL,
    currency text NOT NULL,
    outcome text NOT NULL,
    campaign_state_at_evaluation text NOT NULL,
    configured_budget_minor bigint NOT NULL,
    committed_spend_before_minor bigint NOT NULL,
    resulting_committed_spend_minor bigint NOT NULL,
    resulting_remaining_budget_minor bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT budget_consumption_campaign_fk FOREIGN KEY (campaign_id)
        REFERENCES campaigns(id) ON DELETE RESTRICT,
    CONSTRAINT budget_consumption_key_valid CHECK (octet_length(idempotency_key) BETWEEN 1 AND 128),
    CONSTRAINT budget_consumption_fingerprint_valid CHECK (octet_length(request_fingerprint) = 32),
    CONSTRAINT budget_consumption_amount_positive CHECK (amount_minor > 0),
    CONSTRAINT budget_consumption_currency_supported CHECK (currency IN ('EUR', 'GBP', 'USD')),
    CONSTRAINT budget_consumption_outcome_valid CHECK (
        outcome IN ('APPROVED', 'INSUFFICIENT_BUDGET', 'CAMPAIGN_NOT_ACTIVE')
    ),
    CONSTRAINT budget_consumption_state_valid CHECK (
        campaign_state_at_evaluation IN ('DRAFT', 'ACTIVE', 'PAUSED', 'ENDED')
    ),
    CONSTRAINT budget_consumption_configured_positive CHECK (configured_budget_minor > 0),
    CONSTRAINT budget_consumption_before_valid CHECK (
        committed_spend_before_minor >= 0
        AND committed_spend_before_minor <= configured_budget_minor
    ),
    CONSTRAINT budget_consumption_result_valid CHECK (
        resulting_committed_spend_minor >= 0
        AND resulting_committed_spend_minor <= configured_budget_minor
    ),
    CONSTRAINT budget_consumption_remaining_valid CHECK (
        resulting_remaining_budget_minor >= 0
        AND resulting_remaining_budget_minor = configured_budget_minor - resulting_committed_spend_minor
    ),
    CONSTRAINT budget_consumption_outcome_consistent CHECK (
        (outcome = 'APPROVED'
            AND campaign_state_at_evaluation = 'ACTIVE'
            AND resulting_committed_spend_minor >= committed_spend_before_minor
            AND resulting_committed_spend_minor - committed_spend_before_minor = amount_minor)
        OR
        (outcome = 'INSUFFICIENT_BUDGET'
            AND campaign_state_at_evaluation = 'ACTIVE'
            AND resulting_committed_spend_minor = committed_spend_before_minor
            AND amount_minor > configured_budget_minor - committed_spend_before_minor)
        OR
        (outcome = 'CAMPAIGN_NOT_ACTIVE'
            AND campaign_state_at_evaluation <> 'ACTIVE'
            AND resulting_committed_spend_minor = committed_spend_before_minor)
    ),
    CONSTRAINT budget_consumption_campaign_key_unique UNIQUE (campaign_id, idempotency_key)
);

-- +goose StatementBegin
CREATE FUNCTION reject_budget_consumption_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'budget consumption receipts are append-only' USING ERRCODE = '55000';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER budget_consumption_append_only
    BEFORE UPDATE OR DELETE ON budget_consumption_commands
    FOR EACH ROW EXECUTE FUNCTION reject_budget_consumption_mutation();

-- +goose Down
DROP TABLE budget_consumption_commands;
DROP FUNCTION reject_budget_consumption_mutation();
ALTER TABLE campaigns DROP COLUMN committed_spend_minor;
