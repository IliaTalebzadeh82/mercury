package decision

import (
	"bytes"
	"context"
	"crypto/md5" // Rendezvous ranking compatibility only; never use MD5 for security.
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const explanationLimit = 200

var (
	ErrUnsupportedPlacement = errors.New("unsupported placement")
	ErrDatabase             = errors.New("decision database operation failed")
	ErrInternal             = errors.New("decision internal operation failed")
	lowerUUIDPattern        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	placementPattern        = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	countryPattern          = regexp.MustCompile(`^[A-Z]{2}$`)
)

type ValidationError struct{ Fields map[string]string }

func (e *ValidationError) Error() string { return "ad opportunity validation failed" }

type Opportunity struct {
	ID        string `json:"opportunity_id"`
	Placement string `json:"placement"`
	Country   string `json:"country"`
}

type Selection struct {
	CampaignID      string `json:"campaign_id"`
	CampaignVersion int64  `json:"campaign_version"`
}

type Decision struct {
	DecisionID    string     `json:"decision_id"`
	OpportunityID string     `json:"opportunity_id"`
	Placement     string     `json:"placement"`
	Country       string     `json:"country"`
	Outcome       string     `json:"outcome"`
	Selection     *Selection `json:"selection"`

	EligibleCandidateCount int64         `json:"-"`
	QueryDuration          time.Duration `json:"-"`
}

type Explanation struct {
	CampaignID      string   `json:"campaign_id"`
	CampaignVersion int64    `json:"campaign_version"`
	Eligible        bool     `json:"eligible"`
	Reasons         []string `json:"reasons"`
}

type Diagnostic struct {
	Decision     Decision      `json:"decision"`
	Explanations []Explanation `json:"explanations"`
	Truncated    bool          `json:"truncated"`
}

type candidate struct {
	ID              string
	Version         int64
	State           string
	Placement       string
	CountryTargeted bool
	Score           [md5.Size]byte
}

type Engine struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewEngine(pool *pgxpool.Pool, timeout time.Duration) *Engine {
	return &Engine{pool: pool, timeout: timeout}
}

const selectionSQL = `
	WITH eligible AS (
		SELECT c.id,c.version,decode(md5($1::uuid::text || ':' || c.id::text),'hex') AS ranking_score
		FROM campaign_target_countries AS t
		JOIN campaigns AS c ON c.id=t.campaign_id
		WHERE t.country_code=$3 AND c.state='ACTIVE' AND c.placement_code=$2
	), ranked AS (
		SELECT id,version,count(*) OVER () AS eligible_candidate_count
		FROM eligible
		ORDER BY ranking_score DESC,id ASC
		LIMIT 1
	)
	SELECT p.code,ranked.id::text,ranked.version,COALESCE(ranked.eligible_candidate_count,0)
	FROM placements AS p
	LEFT JOIN ranked ON true
	WHERE p.code=$2`

func (e *Engine) Decide(ctx context.Context, raw Opportunity) (Decision, error) {
	opportunity, err := Normalize(raw)
	if err != nil {
		return Decision{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	result, err := e.queryDecision(ctx, e.pool, opportunity)
	if err != nil {
		return Decision{}, err
	}
	return completeDecision(result)
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (e *Engine) queryDecision(ctx context.Context, database rowQuerier, opportunity Opportunity) (Decision, error) {
	started := time.Now()
	var placement string
	var campaignID pgtype.Text
	var campaignVersion pgtype.Int8
	var count int64
	if err := database.QueryRow(ctx, selectionSQL, opportunity.ID, opportunity.Placement, opportunity.Country).
		Scan(&placement, &campaignID, &campaignVersion, &count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{}, ErrUnsupportedPlacement
		}
		return Decision{}, ErrDatabase
	}
	queryDuration := time.Since(started)
	result := Decision{
		OpportunityID: opportunity.ID, Placement: placement, Country: opportunity.Country,
		Outcome: "NO_FILL", Selection: nil, EligibleCandidateCount: count, QueryDuration: queryDuration,
	}
	if campaignID.Valid {
		result.Outcome = "FILL"
		result.Selection = &Selection{CampaignID: campaignID.String, CampaignVersion: campaignVersion.Int64}
	}
	return result, nil
}

func completeDecision(result Decision) (Decision, error) {
	decisionID, err := randomUUID()
	if err != nil {
		return Decision{}, ErrInternal
	}
	result.DecisionID = decisionID
	return result, nil
}

func (e *Engine) Explain(ctx context.Context, raw Opportunity) (Diagnostic, error) {
	opportunity, err := Normalize(raw)
	if err != nil {
		return Diagnostic{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Diagnostic{}, ErrDatabase
	}
	defer rollback(tx)
	result, err := e.queryDecision(ctx, tx, opportunity)
	if err != nil {
		return Diagnostic{}, err
	}
	candidates, truncated, err := loadDiagnosticCandidates(ctx, tx, opportunity.Country)
	if err != nil {
		return Diagnostic{}, ErrDatabase
	}
	if result.Selection != nil && !containsCandidate(candidates, result.Selection.CampaignID) {
		selected, err := loadDiagnosticCandidate(ctx, tx, result.Selection.CampaignID, opportunity.Country)
		if err != nil {
			return Diagnostic{}, ErrDatabase
		}
		if len(candidates) == explanationLimit {
			candidates[len(candidates)-1] = selected
		} else {
			candidates = append(candidates, selected)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Diagnostic{}, ErrDatabase
	}
	result, err = completeDecision(result)
	if err != nil {
		return Diagnostic{}, err
	}
	explanations := make([]Explanation, 0, len(candidates))
	for _, item := range candidates {
		reasons := rejectionReasons(item, opportunity)
		explanations = append(explanations, Explanation{
			CampaignID: item.ID, CampaignVersion: item.Version, Eligible: len(reasons) == 0, Reasons: reasons,
		})
	}
	return Diagnostic{Decision: result, Explanations: explanations, Truncated: truncated}, nil
}

func loadDiagnosticCandidates(ctx context.Context, tx pgx.Tx, country string) ([]candidate, bool, error) {
	rows, err := tx.Query(ctx, `
		SELECT c.id::text,c.version,c.state,c.placement_code,
			EXISTS(SELECT 1 FROM campaign_target_countries AS t WHERE t.campaign_id=c.id AND t.country_code=$1)
		FROM campaigns AS c ORDER BY c.id LIMIT $2`, country, explanationLimit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	values := make([]candidate, 0, explanationLimit)
	truncated := false
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.ID, &value.Version, &value.State, &value.Placement, &value.CountryTargeted); err != nil {
			return nil, false, err
		}
		if len(values) == explanationLimit {
			truncated = true
			continue
		}
		values = append(values, value)
	}
	return values, truncated, rows.Err()
}

func loadDiagnosticCandidate(ctx context.Context, tx pgx.Tx, id, country string) (candidate, error) {
	var value candidate
	err := tx.QueryRow(ctx, `
		SELECT c.id::text,c.version,c.state,c.placement_code,
			EXISTS(SELECT 1 FROM campaign_target_countries AS t WHERE t.campaign_id=c.id AND t.country_code=$2)
		FROM campaigns AS c WHERE c.id=$1`, id, country).
		Scan(&value.ID, &value.Version, &value.State, &value.Placement, &value.CountryTargeted)
	return value, err
}

func containsCandidate(values []candidate, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func rejectionReasons(value candidate, opportunity Opportunity) []string {
	reasons := make([]string, 0, 3)
	if value.State != "ACTIVE" {
		reasons = append(reasons, "campaign_not_active")
	}
	if value.Placement != opportunity.Placement {
		reasons = append(reasons, "placement_mismatch")
	}
	if !value.CountryTargeted {
		reasons = append(reasons, "country_not_targeted")
	}
	return reasons
}

func Normalize(value Opportunity) (Opportunity, error) {
	fields := make(map[string]string)
	if !lowerUUIDPattern.MatchString(value.ID) || value.ID == "00000000-0000-0000-0000-000000000000" {
		fields["opportunity_id"] = "must be a canonical lowercase non-nil UUID"
	}
	value.Placement = strings.ToLower(strings.TrimSpace(value.Placement))
	if !placementPattern.MatchString(value.Placement) {
		fields["placement"] = "must start with a letter and contain only lowercase letters, digits, or underscores"
	}
	value.Country = strings.ToUpper(strings.TrimSpace(value.Country))
	if !countryPattern.MatchString(value.Country) {
		fields["country"] = "must contain exactly two ASCII letters"
	}
	if len(fields) > 0 {
		return Opportunity{}, &ValidationError{Fields: fields}
	}
	return value, nil
}

func score(opportunityID, campaignID string) [md5.Size]byte {
	return md5.Sum([]byte(opportunityID + ":" + campaignID))
}

func choose(opportunityID string, values []candidate) (candidate, bool) {
	if len(values) == 0 {
		return candidate{}, false
	}
	best := values[0]
	best.Score = score(opportunityID, best.ID)
	for _, current := range values[1:] {
		current.Score = score(opportunityID, current.ID)
		if better(current, best) {
			best = current
		}
	}
	return best, true
}

func better(current, best candidate) bool {
	comparison := bytes.Compare(current.Score[:], best.Score[:])
	return comparison > 0 || comparison == 0 && current.ID < best.ID
}

func randomUUID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
