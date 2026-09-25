package campaign

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func getCampaign(ctx context.Context, database queryer, id string, forUpdate bool) (Campaign, error) {
	query := `
		SELECT id::text,advertiser_id::text,name,state,placement_code,budget_amount_minor,committed_spend_minor,currency,version,created_at,updated_at
		FROM campaigns WHERE id=$1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var value Campaign
	err := database.QueryRow(ctx, query, id).Scan(campaignDestinations(&value)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	if err != nil {
		return Campaign{}, ErrDatabase
	}
	value.Countries, err = loadCountries(ctx, database, value.ID)
	if err != nil {
		return Campaign{}, ErrDatabase
	}
	return value, nil
}

func campaignDestinations(value *Campaign) []any {
	return []any{
		&value.ID, &value.AdvertiserID, &value.Name, &value.State, &value.PlacementCode,
		&value.BudgetAmountMinor, &value.CommittedSpendMinor, &value.Currency, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	}
}

func loadCountries(ctx context.Context, database queryer, campaignID string) ([]string, error) {
	rows, err := database.Query(ctx, `
		SELECT country_code FROM campaign_target_countries
		WHERE campaign_id=$1 ORDER BY country_code`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func validatePlacement(ctx context.Context, database queryer, code string) error {
	var exists bool
	if err := database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM placements WHERE code=$1)`, code).Scan(&exists); err != nil {
		return ErrDatabase
	}
	if !exists {
		return &ValidationError{Fields: map[string]string{"placement_code": "is not a supported placement"}}
	}
	return nil
}

type mutation func(context.Context, pgx.Tx, *Campaign) (bool, error)

func (s *Store) mutate(ctx context.Context, id string, expectedVersion int64, change mutation) (Campaign, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, ErrDatabase
	}
	defer rollback(tx)

	value, err := getCampaign(ctx, tx, id, true)
	if err != nil {
		return Campaign{}, err
	}
	if value.Version != expectedVersion {
		return Campaign{}, ErrPreconditionFailed
	}
	changed, err := change(ctx, tx, &value)
	if err != nil {
		return Campaign{}, err
	}
	if changed {
		value, err = getCampaign(ctx, tx, id, false)
		if err != nil {
			return Campaign{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Campaign{}, ErrDatabase
	}
	return value, nil
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (s *Store) UpdateName(ctx context.Context, id string, expectedVersion int64, rawName string) (Campaign, error) {
	name := stringsTrimmedName(rawName)
	if name == "" {
		return Campaign{}, &ValidationError{Fields: map[string]string{"name": "must contain between 1 and 200 characters"}}
	}
	return s.mutate(ctx, id, expectedVersion, func(ctx context.Context, tx pgx.Tx, value *Campaign) (bool, error) {
		if value.State == Ended {
			return false, &ConflictError{Code: "campaign_ended"}
		}
		if value.Name == name {
			return false, nil
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET name=$2,updated_at=now(),version=version+1 WHERE id=$1`, id, name); err != nil {
			return false, ErrDatabase
		}
		return true, nil
	})
}

func (s *Store) UpdateBudget(ctx context.Context, id string, expectedVersion, amount int64) (Campaign, error) {
	if amount <= 0 {
		return Campaign{}, &ValidationError{Fields: map[string]string{"configured_amount_minor": "must be a positive integer"}}
	}
	return s.mutate(ctx, id, expectedVersion, func(ctx context.Context, tx pgx.Tx, value *Campaign) (bool, error) {
		if value.State == Ended {
			return false, &ConflictError{Code: "campaign_ended"}
		}
		if amount < value.CommittedSpendMinor {
			return false, &ConflictError{Code: "configured_budget_below_committed_spend"}
		}
		if value.State == Active && amount < value.BudgetAmountMinor {
			return false, &ConflictError{Code: "active_budget_cannot_decrease"}
		}
		if value.BudgetAmountMinor == amount {
			return false, nil
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET budget_amount_minor=$2,updated_at=now(),version=version+1 WHERE id=$1`, id, amount); err != nil {
			return false, ErrDatabase
		}
		return true, nil
	})
}

func (s *Store) UpdatePlacement(ctx context.Context, id string, expectedVersion int64, rawCode string) (Campaign, error) {
	code := normalizePlacement(rawCode)
	if code == "" {
		return Campaign{}, &ValidationError{Fields: map[string]string{"placement_code": "is required"}}
	}
	return s.mutate(ctx, id, expectedVersion, func(ctx context.Context, tx pgx.Tx, value *Campaign) (bool, error) {
		if value.State == Ended {
			return false, &ConflictError{Code: "campaign_ended"}
		}
		if value.State == Active {
			return false, &ConflictError{Code: "placement_immutable_while_active"}
		}
		if err := validatePlacement(ctx, tx, code); err != nil {
			return false, err
		}
		if value.PlacementCode == code {
			return false, nil
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET placement_code=$2,updated_at=now(),version=version+1 WHERE id=$1`, id, code); err != nil {
			return false, ErrDatabase
		}
		return true, nil
	})
}

func (s *Store) UpdateTargeting(ctx context.Context, id string, expectedVersion int64, rawCountries []string) (Campaign, error) {
	fields := make(map[string]string)
	countries := normalizeCountries(rawCountries, fields)
	if len(fields) > 0 {
		return Campaign{}, &ValidationError{Fields: fields}
	}
	return s.mutate(ctx, id, expectedVersion, func(ctx context.Context, tx pgx.Tx, value *Campaign) (bool, error) {
		if value.State == Ended {
			return false, &ConflictError{Code: "campaign_ended"}
		}
		if value.State == Active {
			return false, &ConflictError{Code: "targeting_immutable_while_active"}
		}
		if slices.Equal(value.Countries, countries) {
			return false, nil
		}
		if _, err := tx.Exec(ctx, `DELETE FROM campaign_target_countries WHERE campaign_id=$1`, id); err != nil {
			return false, ErrDatabase
		}
		for _, country := range countries {
			if _, err := tx.Exec(ctx, `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,$2)`, id, country); err != nil {
				return false, ErrDatabase
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET updated_at=now(),version=version+1 WHERE id=$1`, id); err != nil {
			return false, ErrDatabase
		}
		return true, nil
	})
}

func (s *Store) Activate(ctx context.Context, id string, expectedVersion int64) (Campaign, error) {
	return s.transition(ctx, id, expectedVersion, Draft, Active, true)
}

func (s *Store) Pause(ctx context.Context, id string, expectedVersion int64) (Campaign, error) {
	return s.transition(ctx, id, expectedVersion, Active, Paused, false)
}

func (s *Store) Resume(ctx context.Context, id string, expectedVersion int64) (Campaign, error) {
	return s.transition(ctx, id, expectedVersion, Paused, Active, true)
}

func (s *Store) End(ctx context.Context, id string, expectedVersion int64) (Campaign, error) {
	return s.mutate(ctx, id, expectedVersion, func(ctx context.Context, tx pgx.Tx, value *Campaign) (bool, error) {
		if value.State == Ended {
			return false, &ConflictError{Code: "invalid_lifecycle_transition"}
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET state='ENDED',updated_at=now(),version=version+1 WHERE id=$1`, id); err != nil {
			return false, ErrDatabase
		}
		return true, nil
	})
}

func (s *Store) transition(ctx context.Context, id string, expectedVersion int64, from, to State, validateActivation bool) (Campaign, error) {
	return s.mutate(ctx, id, expectedVersion, func(ctx context.Context, tx pgx.Tx, value *Campaign) (bool, error) {
		if value.State != from {
			return false, &ConflictError{Code: "invalid_lifecycle_transition"}
		}
		if validateActivation {
			if value.BudgetAmountMinor <= 0 || len(value.Countries) == 0 {
				return false, &ConflictError{Code: "activation_requirements_not_met"}
			}
			if err := validatePlacement(ctx, tx, value.PlacementCode); err != nil {
				return false, &ConflictError{Code: "activation_requirements_not_met"}
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET state=$2,updated_at=now(),version=version+1 WHERE id=$1`, id, to); err != nil {
			return false, ErrDatabase
		}
		return true, nil
	})
}
