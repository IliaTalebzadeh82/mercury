package budget

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Outcome string

const (
	Approved           Outcome = "APPROVED"
	InsufficientBudget Outcome = "INSUFFICIENT_BUDGET"
	CampaignNotActive  Outcome = "CAMPAIGN_NOT_ACTIVE"
)

var (
	ErrNotFound            = errors.New("campaign not found")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with an existing budget consumption")
	ErrCurrencyMismatch    = errors.New("consumption currency does not match campaign currency")
	ErrDatabase            = errors.New("budget database operation failed")
	ErrOutcomeUnknown      = errors.New("budget consumption commit outcome is unknown")
	uuidPattern            = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

type ValidationError struct{ Fields map[string]string }

func (e *ValidationError) Error() string { return "budget consumption validation failed" }

type Account struct {
	CampaignID            string
	ConfiguredAmountMinor int64
	CommittedSpendMinor   int64
	RemainingAmountMinor  int64
	Currency              string
}

type Command struct {
	CampaignID     string
	AmountMinor    int64
	Currency       string
	IdempotencyKey string
}

type Result struct {
	ConsumptionID       string
	CampaignID          string
	Outcome             Outcome
	AmountMinor         int64
	Currency            string
	CampaignState       string
	ConfiguredMinor     int64
	CommittedBefore     int64
	ResultingCommitted  int64
	ResultingRemaining  int64
	CreatedAt           time.Time
	Replayed            bool
	CampaignLockWait    time.Duration
	TransactionDuration time.Duration
}

type Store struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewStore(pool *pgxpool.Pool, timeout time.Duration) *Store {
	return &Store{pool: pool, timeout: timeout}
}

func (s *Store) GetAccount(ctx context.Context, campaignID string) (Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var value Account
	err := s.pool.QueryRow(ctx, `
		SELECT id::text,budget_amount_minor,committed_spend_minor,
			budget_amount_minor-committed_spend_minor,currency
		FROM campaigns WHERE id=$1`, campaignID).Scan(
		&value.CampaignID, &value.ConfiguredAmountMinor, &value.CommittedSpendMinor,
		&value.RemainingAmountMinor, &value.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, ErrDatabase
	}
	return value, nil
}

func (s *Store) Consume(ctx context.Context, raw Command) (Result, error) {
	command, validationErr := normalize(raw)
	if validationErr != nil {
		return Result{}, validationErr
	}
	fingerprint := commandFingerprint(command)
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, ErrDatabase
	}
	defer rollback(tx)
	started := time.Now()

	lockStarted := time.Now()
	var state, currency string
	var configured, committed int64
	err = tx.QueryRow(ctx, `
		SELECT state,currency,budget_amount_minor,committed_spend_minor
		FROM campaigns WHERE id=$1 FOR UPDATE`, command.CampaignID).
		Scan(&state, &currency, &configured, &committed)
	lockWait := time.Since(lockStarted)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, ErrDatabase
	}

	stored, storedFingerprint, err := loadReceipt(ctx, tx, command.CampaignID, command.IdempotencyKey)
	if err == nil {
		if !equalBytes(storedFingerprint, fingerprint[:]) {
			return Result{}, ErrIdempotencyConflict
		}
		stored.Replayed = true
		stored.CampaignLockWait = lockWait
		stored.TransactionDuration = time.Since(started)
		return stored, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrDatabase
	}
	if currency != command.Currency {
		return Result{}, ErrCurrencyMismatch
	}

	result := Result{
		CampaignID: command.CampaignID, AmountMinor: command.AmountMinor, Currency: currency,
		CampaignState: state, ConfiguredMinor: configured, CommittedBefore: committed,
		ResultingCommitted: committed, CampaignLockWait: lockWait,
	}
	remaining := configured - committed
	switch {
	case state != "ACTIVE":
		result.Outcome = CampaignNotActive
	case command.AmountMinor > remaining:
		result.Outcome = InsufficientBudget
	default:
		result.Outcome = Approved
		result.ResultingCommitted = committed + command.AmountMinor
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET committed_spend_minor=$2 WHERE id=$1`, command.CampaignID, result.ResultingCommitted); err != nil {
			return Result{}, ErrDatabase
		}
	}
	result.ResultingRemaining = configured - result.ResultingCommitted

	err = tx.QueryRow(ctx, `
		INSERT INTO budget_consumption_commands
			(campaign_id,idempotency_key,request_fingerprint,amount_minor,currency,outcome,
			 campaign_state_at_evaluation,configured_budget_minor,committed_spend_before_minor,
			 resulting_committed_spend_minor,resulting_remaining_budget_minor)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id::text,created_at`,
		command.CampaignID, command.IdempotencyKey, fingerprint[:], command.AmountMinor, currency,
		result.Outcome, state, configured, committed, result.ResultingCommitted, result.ResultingRemaining,
	).Scan(&result.ConsumptionID, &result.CreatedAt)
	if err != nil {
		return Result{}, ErrDatabase
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, classifyCommitError(err)
	}
	result.TransactionDuration = time.Since(started)
	return result, nil
}

func classifyCommitError(err error) error {
	if errors.Is(err, pgx.ErrTxCommitRollback) {
		return ErrDatabase
	}
	return ErrOutcomeUnknown
}

func normalize(command Command) (Command, *ValidationError) {
	fields := make(map[string]string)
	command.CampaignID = strings.ToLower(strings.TrimSpace(command.CampaignID))
	if !uuidPattern.MatchString(command.CampaignID) || command.CampaignID == "00000000-0000-0000-0000-000000000000" {
		fields["campaign_id"] = "must be a canonical non-nil UUID"
	}
	if command.AmountMinor <= 0 {
		fields["amount_minor"] = "must be a positive integer"
	}
	command.Currency = strings.ToUpper(strings.TrimSpace(command.Currency))
	if command.Currency != "EUR" && command.Currency != "GBP" && command.Currency != "USD" {
		fields["currency"] = "must be one of EUR, GBP, or USD"
	}
	if len(command.IdempotencyKey) < 1 || len(command.IdempotencyKey) > 128 {
		fields["idempotency_key"] = "must contain between 1 and 128 bytes"
	}
	if len(fields) > 0 {
		return Command{}, &ValidationError{Fields: fields}
	}
	return command, nil
}

func commandFingerprint(command Command) [sha256.Size]byte {
	encoded, err := json.Marshal(struct {
		CampaignID  string `json:"campaign_id"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}{command.CampaignID, command.AmountMinor, command.Currency})
	if err != nil {
		panic(err)
	}
	return sha256.Sum256(encoded)
}

func loadReceipt(ctx context.Context, tx pgx.Tx, campaignID, key string) (Result, []byte, error) {
	var result Result
	var fingerprint []byte
	err := tx.QueryRow(ctx, `
		SELECT id::text,campaign_id::text,outcome,amount_minor,currency,campaign_state_at_evaluation,
			configured_budget_minor,committed_spend_before_minor,resulting_committed_spend_minor,
			resulting_remaining_budget_minor,created_at,request_fingerprint
		FROM budget_consumption_commands
		WHERE campaign_id=$1 AND idempotency_key=$2`, campaignID, key).Scan(
		&result.ConsumptionID, &result.CampaignID, &result.Outcome, &result.AmountMinor,
		&result.Currency, &result.CampaignState, &result.ConfiguredMinor, &result.CommittedBefore,
		&result.ResultingCommitted, &result.ResultingRemaining, &result.CreatedAt, &fingerprint,
	)
	return result, fingerprint, err
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

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
