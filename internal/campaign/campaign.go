package campaign

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type State string

const (
	Draft  State = "DRAFT"
	Active State = "ACTIVE"
	Paused State = "PAUSED"
	Ended  State = "ENDED"
)

var (
	ErrNotFound            = errors.New("campaign not found")
	ErrAdvertiserNotFound  = errors.New("advertiser not found")
	ErrPreconditionFailed  = errors.New("campaign version precondition failed")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with an existing campaign")
	ErrDatabase            = errors.New("campaign database operation failed")
	countryPattern         = regexp.MustCompile(`^[A-Z]{2}$`)
)

type ValidationError struct{ Fields map[string]string }

func (e *ValidationError) Error() string { return "campaign validation failed" }

type ConflictError struct{ Code string }

func (e *ConflictError) Error() string { return e.Code }

type Placement struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type Campaign struct {
	ID                string    `json:"id"`
	AdvertiserID      string    `json:"advertiser_id"`
	Name              string    `json:"name"`
	State             State     `json:"state"`
	PlacementCode     string    `json:"placement_code"`
	BudgetAmountMinor int64     `json:"-"`
	Currency          string    `json:"-"`
	Countries         []string  `json:"-"`
	Version           int64     `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateCommand struct {
	AdvertiserID   string
	Name           string
	PlacementCode  string
	AmountMinor    int64
	Currency       string
	Countries      []string
	IdempotencyKey string
}

type CreateResult struct {
	Campaign Campaign
	Replayed bool
}

type Store struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewStore(pool *pgxpool.Pool, timeout time.Duration) *Store {
	return &Store{pool: pool, timeout: timeout}
}

func (s *Store) Create(ctx context.Context, command CreateCommand) (CreateResult, error) {
	normalized, validationErr := normalizeCreate(command)
	if validationErr != nil {
		return CreateResult{}, validationErr
	}
	fingerprint := sha256.Sum256(mustJSON(struct {
		AdvertiserID  string   `json:"advertiser_id"`
		Name          string   `json:"name"`
		PlacementCode string   `json:"placement_code"`
		AmountMinor   int64    `json:"amount_minor"`
		Currency      string   `json:"currency"`
		Countries     []string `json:"countries"`
	}{normalized.AdvertiserID, normalized.Name, normalized.PlacementCode, normalized.AmountMinor, normalized.Currency, normalized.Countries}))

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateResult{}, ErrDatabase
	}
	defer rollback(tx)

	var advertiserExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM advertisers WHERE id=$1)`, normalized.AdvertiserID).Scan(&advertiserExists); err != nil {
		return CreateResult{}, ErrDatabase
	}
	if !advertiserExists {
		return CreateResult{}, ErrAdvertiserNotFound
	}
	if err := validatePlacement(ctx, tx, normalized.PlacementCode); err != nil {
		return CreateResult{}, err
	}

	var value Campaign
	err = tx.QueryRow(ctx, `
		INSERT INTO campaigns
			(advertiser_id,name,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (advertiser_id,creation_idempotency_key) DO NOTHING
		RETURNING id::text,advertiser_id::text,name,state,placement_code,budget_amount_minor,currency,version,created_at,updated_at`,
		normalized.AdvertiserID, normalized.Name, normalized.PlacementCode, normalized.AmountMinor,
		normalized.Currency, normalized.IdempotencyKey, fingerprint[:]).Scan(campaignDestinations(&value)...)
	if err == nil {
		for _, country := range normalized.Countries {
			if _, err := tx.Exec(ctx, `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,$2)`, value.ID, country); err != nil {
				return CreateResult{}, ErrDatabase
			}
		}
		value.Countries = normalized.Countries
		if err := tx.Commit(ctx); err != nil {
			return CreateResult{}, ErrDatabase
		}
		return CreateResult{Campaign: value}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return CreateResult{}, ErrDatabase
	}

	var storedFingerprint []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text,advertiser_id::text,name,state,placement_code,budget_amount_minor,currency,version,created_at,updated_at,creation_request_fingerprint
		FROM campaigns WHERE advertiser_id=$1 AND creation_idempotency_key=$2`, normalized.AdvertiserID, normalized.IdempotencyKey).Scan(
		append(campaignDestinations(&value), &storedFingerprint)...,
	)
	if err != nil {
		return CreateResult{}, ErrDatabase
	}
	if !equalBytes(storedFingerprint, fingerprint[:]) {
		return CreateResult{}, ErrIdempotencyConflict
	}
	value.Countries, err = loadCountries(ctx, tx, value.ID)
	if err != nil {
		return CreateResult{}, ErrDatabase
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateResult{}, ErrDatabase
	}
	return CreateResult{Campaign: value, Replayed: true}, nil
}

func (s *Store) Get(ctx context.Context, id string) (Campaign, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return getCampaign(ctx, s.pool, id, false)
}

func (s *Store) List(ctx context.Context, advertiserID string, limit, offset int) ([]Campaign, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		SELECT id::text,advertiser_id::text,name,state,placement_code,budget_amount_minor,currency,version,created_at,updated_at,
			ARRAY(SELECT country_code FROM campaign_target_countries t WHERE t.campaign_id=campaigns.id ORDER BY country_code)
		FROM campaigns WHERE advertiser_id=$1
		ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, advertiserID, limit, offset)
	if err != nil {
		return nil, ErrDatabase
	}
	defer rows.Close()
	values := make([]Campaign, 0)
	for rows.Next() {
		var value Campaign
		if err := rows.Scan(append(campaignDestinations(&value), &value.Countries)...); err != nil {
			return nil, ErrDatabase
		}
		values = append(values, value)
	}
	if rows.Err() != nil {
		return nil, ErrDatabase
	}
	return values, nil
}

func (s *Store) ListPlacements(ctx context.Context) ([]Placement, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	rows, err := s.pool.Query(ctx, `SELECT code,display_name FROM placements ORDER BY code`)
	if err != nil {
		return nil, ErrDatabase
	}
	defer rows.Close()
	values := make([]Placement, 0, 3)
	for rows.Next() {
		var value Placement
		if err := rows.Scan(&value.Code, &value.DisplayName); err != nil {
			return nil, ErrDatabase
		}
		values = append(values, value)
	}
	if rows.Err() != nil {
		return nil, ErrDatabase
	}
	return values, nil
}

func normalizeCreate(command CreateCommand) (CreateCommand, *ValidationError) {
	fields := make(map[string]string)
	command.Name = strings.TrimSpace(command.Name)
	if command.Name == "" || utf8.RuneCountInString(command.Name) > 200 {
		fields["name"] = "must contain between 1 and 200 characters"
	}
	command.PlacementCode = strings.ToLower(strings.TrimSpace(command.PlacementCode))
	if command.PlacementCode == "" {
		fields["placement_code"] = "is required"
	}
	if command.AmountMinor <= 0 {
		fields["configured_amount_minor"] = "must be a positive integer"
	}
	command.Currency = strings.ToUpper(strings.TrimSpace(command.Currency))
	if command.Currency != "EUR" && command.Currency != "GBP" && command.Currency != "USD" {
		fields["currency"] = "must be one of EUR, GBP, or USD"
	}
	command.Countries = normalizeCountries(command.Countries, fields)
	if len(command.IdempotencyKey) < 1 || len(command.IdempotencyKey) > 128 {
		fields["idempotency_key"] = "must contain between 1 and 128 bytes"
	}
	if len(fields) > 0 {
		return CreateCommand{}, &ValidationError{Fields: fields}
	}
	return command, nil
}

func normalizeCountries(raw []string, fields map[string]string) []string {
	unique := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		country := strings.ToUpper(strings.TrimSpace(item))
		if !countryPattern.MatchString(country) {
			fields["countries"] = "must contain only two-letter country codes"
			continue
		}
		unique[country] = struct{}{}
	}
	values := make([]string, 0, len(unique))
	for country := range unique {
		values = append(values, country)
	}
	sort.Strings(values)
	if len(values) == 0 {
		fields["countries"] = "must contain at least one country code"
	}
	return values
}

func stringsTrimmedName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		return ""
	}
	return name
}

func normalizePlacement(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func mustJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}
