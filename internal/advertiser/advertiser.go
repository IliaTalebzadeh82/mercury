package advertiser

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("advertiser not found")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with an existing advertiser")
	ErrDatabase            = errors.New("advertiser database operation failed")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "advertiser validation failed" }

type Advertiser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCommand struct {
	Name           string
	IdempotencyKey string
}

type CreateResult struct {
	Advertiser Advertiser
	Replayed   bool
}

type Store struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewStore(pool *pgxpool.Pool, timeout time.Duration) *Store {
	return &Store{pool: pool, timeout: timeout}
}

func (s *Store) Create(ctx context.Context, command CreateCommand) (CreateResult, error) {
	name, validationErr := normalizeName(command.Name)
	if validationErr != nil {
		return CreateResult{}, validationErr
	}
	if err := validateKey(command.IdempotencyKey); err != nil {
		return CreateResult{}, err
	}

	fingerprint := sha256.Sum256(mustJSON(struct {
		Name string `json:"name"`
	}{Name: name}))

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateResult{}, ErrDatabase
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.Background(), time.Second)
		defer cancelRollback()
		_ = tx.Rollback(rollbackCtx)
	}()

	var created Advertiser
	err = tx.QueryRow(ctx, `
		INSERT INTO advertisers (name, creation_idempotency_key, creation_request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (creation_idempotency_key) DO NOTHING
		RETURNING id::text, name, created_at`, name, command.IdempotencyKey, fingerprint[:]).Scan(
		&created.ID, &created.Name, &created.CreatedAt,
	)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return CreateResult{}, ErrDatabase
		}
		return CreateResult{Advertiser: created}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return CreateResult{}, ErrDatabase
	}

	var storedFingerprint []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, name, created_at, creation_request_fingerprint
		FROM advertisers WHERE creation_idempotency_key = $1`, command.IdempotencyKey).Scan(
		&created.ID, &created.Name, &created.CreatedAt, &storedFingerprint,
	)
	if err != nil {
		return CreateResult{}, ErrDatabase
	}
	if !equalBytes(storedFingerprint, fingerprint[:]) {
		return CreateResult{}, ErrIdempotencyConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateResult{}, ErrDatabase
	}
	return CreateResult{Advertiser: created, Replayed: true}, nil
}

func (s *Store) Get(ctx context.Context, id string) (Advertiser, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var value Advertiser
	err := s.pool.QueryRow(ctx, `SELECT id::text, name, created_at FROM advertisers WHERE id = $1`, id).Scan(
		&value.ID, &value.Name, &value.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Advertiser{}, ErrNotFound
	}
	if err != nil {
		return Advertiser{}, ErrDatabase
	}
	return value, nil
}

func (s *Store) List(ctx context.Context) ([]Advertiser, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, created_at FROM advertisers
		ORDER BY created_at DESC, id DESC LIMIT 100`)
	if err != nil {
		return nil, ErrDatabase
	}
	defer rows.Close()
	values := make([]Advertiser, 0)
	for rows.Next() {
		var value Advertiser
		if err := rows.Scan(&value.ID, &value.Name, &value.CreatedAt); err != nil {
			return nil, ErrDatabase
		}
		values = append(values, value)
	}
	if rows.Err() != nil {
		return nil, ErrDatabase
	}
	return values, nil
}

func normalizeName(raw string) (string, *ValidationError) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		return "", &ValidationError{Fields: map[string]string{"name": "must contain between 1 and 200 characters"}}
	}
	return name, nil
}

func validateKey(key string) *ValidationError {
	if len(key) < 1 || len(key) > 128 {
		return &ValidationError{Fields: map[string]string{"idempotency_key": "must contain between 1 and 128 bytes"}}
	}
	return nil
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
