package atlas

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const contributionSchemaVersion = 1

// ContributionPackage is a self-describing, offline Community Atlas artifact.
// Location provenance remains attached to every row; the package does not make
// a claim that any contributor-supplied location is authoritative.
type ContributionPackage struct {
	SchemaVersion int       `json:"schema_version"`
	Catalog       Catalog   `json:"catalog"`
	GeneratedAt   time.Time `json:"generated_at"`
	ReviewStatus  string    `json:"review_status"`
}

// NewContributionPackage wraps a validated catalog for community review.
func NewContributionPackage(catalog Catalog, generatedAt time.Time) (ContributionPackage, error) {
	if err := catalog.Validate(); err != nil {
		return ContributionPackage{}, err
	}
	if generatedAt.IsZero() {
		return ContributionPackage{}, fmt.Errorf("contribution generated_at is required")
	}
	return ContributionPackage{SchemaVersion: contributionSchemaVersion, Catalog: catalog, GeneratedAt: generatedAt.UTC(), ReviewStatus: "unreviewed"}, nil
}

// Validate checks the package envelope and its embedded catalog.
func (p ContributionPackage) Validate() error {
	if p.SchemaVersion != contributionSchemaVersion {
		return fmt.Errorf("unsupported contribution schema version %d", p.SchemaVersion)
	}
	if p.GeneratedAt.IsZero() {
		return fmt.Errorf("contribution generated_at is required")
	}
	if p.ReviewStatus == "" {
		return fmt.Errorf("contribution review_status is required")
	}
	return p.Catalog.Validate()
}

// SaveContribution writes an owner-readable package using an atomic replacement.
func SaveContribution(path string, contribution ContributionPackage) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("contribution path is required")
	}
	if err := contribution.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(contribution, "", "  ")
	if err != nil {
		return fmt.Errorf("encode contribution: %w", err)
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve contribution path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create contribution directory: %w", err)
	}
	temporary := target + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write contribution: %w", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace contribution: %w", err)
	}
	return nil
}

// LoadContribution reads and validates a local community package.
func LoadContribution(path string) (ContributionPackage, error) {
	file, err := os.Open(path)
	if err != nil {
		return ContributionPackage{}, err
	}
	defer file.Close()
	var contribution ContributionPackage
	if err := json.NewDecoder(io.LimitReader(file, 32<<20)).Decode(&contribution); err != nil {
		return ContributionPackage{}, fmt.Errorf("decode contribution: %w", err)
	}
	if err := contribution.Validate(); err != nil {
		return ContributionPackage{}, err
	}
	return contribution, nil
}
