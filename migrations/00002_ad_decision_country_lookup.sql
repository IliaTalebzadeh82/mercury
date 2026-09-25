-- +goose Up
CREATE INDEX campaign_target_countries_country_campaign_idx
    ON campaign_target_countries (country_code, campaign_id);

-- +goose Down
DROP INDEX campaign_target_countries_country_campaign_idx;
