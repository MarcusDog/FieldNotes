package inspection

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type retryRecord struct {
	Fingerprint string `json:"fingerprint"`
	Issue       Issue  `json:"issue"`
}
type diskState struct {
	Version     int                    `json:"version"`
	Issues      []Issue                `json:"issues"`
	Idempotency map[string]retryRecord `json:"idempotency"`
}

// Store serializes mutations and publishes memory only after an atomic disk write.
// One process must own the data file; it is not a multi-process database.
type Store struct {
	mu    sync.RWMutex
	path  string
	state diskState
}

func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("data file path is empty")
	}
	s := &Store{path: path, state: diskState{Version: 1, Issues: []Issue{}, Idempotency: map[string]retryRecord{}}}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open data: %w", err)
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s.state); err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, errors.New("unexpected trailing data in data file")
	}
	if err := validateState(s.state); err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}
	return s, nil
}
func validateIssue(i Issue) error {
	if len(i.ID) != 32 {
		return errors.New("invalid issue id")
	}
	if _, err := hex.DecodeString(i.ID); err != nil {
		return errors.New("invalid issue id")
	}
	if !validCreate(CreateIssue{i.Title, i.Description, i.Priority, i.Position}) || !validStatus(i.Status) {
		return errors.New("invalid issue fields")
	}
	for _, v := range []string{i.CreatedAt, i.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339Nano, v); err != nil || !strings.HasSuffix(v, "Z") {
			return errors.New("invalid issue timestamp")
		}
	}
	created, _ := time.Parse(time.RFC3339Nano, i.CreatedAt)
	updated, _ := time.Parse(time.RFC3339Nano, i.UpdatedAt)
	if updated.Before(created) {
		return errors.New("updated time precedes creation")
	}
	return nil
}
func validateState(state diskState) error {
	if state.Version != 1 || state.Issues == nil || state.Idempotency == nil {
		return errors.New("unsupported or incomplete data format")
	}
	seen := map[string]bool{}
	for _, issue := range state.Issues {
		if err := validateIssue(issue); err != nil {
			return err
		}
		if seen[issue.ID] {
			return errors.New("duplicate issue id")
		}
		seen[issue.ID] = true
	}
	for key, record := range state.Idempotency {
		if !validKey(key) || len(record.Fingerprint) != 64 || !seen[record.Issue.ID] {
			return errors.New("invalid idempotency record")
		}
		if _, err := hex.DecodeString(record.Fingerprint); err != nil {
			return err
		}
		if err := validateIssue(record.Issue); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) List() []Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Issue, len(s.state.Issues))
	copy(result, s.state.Issues)
	return result
}
func (s *Store) Create(input CreateIssue, key string) (Issue, error) {
	if !validCreate(input) || (key != "" && !validKey(key)) {
		return Issue{}, errors.New("invalid create input")
	}
	payload, _ := json.Marshal(input)
	sum := sha256.Sum256(payload)
	fingerprint := hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if record, ok := s.state.Idempotency[key]; key != "" && ok {
		if record.Fingerprint != fingerprint {
			return Issue{}, ErrConflict
		}
		return record.Issue, nil
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return Issue{}, fmt.Errorf("generate id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	issue := Issue{ID: hex.EncodeToString(random), Title: strings.TrimSpace(input.Title), Description: input.Description, Priority: input.Priority, Status: "open", Position: input.Position, CreatedAt: now, UpdatedAt: now}
	next := s.copyState()
	next.Issues = append([]Issue{issue}, next.Issues...)
	if key != "" {
		next.Idempotency[key] = retryRecord{fingerprint, issue}
	}
	if err := s.persist(next); err != nil {
		return Issue{}, err
	}
	s.state = next
	return issue, nil
}
func (s *Store) UpdateStatus(id, status string) (Issue, error) {
	if !validStatus(status) {
		return Issue{}, errors.New("invalid status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, issue := range s.state.Issues {
		if issue.ID == id {
			issue.Status = status
			now := time.Now().UTC()
			previous, _ := time.Parse(time.RFC3339Nano, issue.UpdatedAt)
			if now.Before(previous) {
				now = previous
			}
			issue.UpdatedAt = now.Format(time.RFC3339Nano)
			next := s.copyState()
			next.Issues[index] = issue
			if err := s.persist(next); err != nil {
				return Issue{}, err
			}
			s.state = next
			return issue, nil
		}
	}
	return Issue{}, ErrNotFound
}
func (s *Store) copyState() diskState {
	next := diskState{Version: 1, Issues: append([]Issue{}, s.state.Issues...), Idempotency: make(map[string]retryRecord, len(s.state.Idempotency))}
	for k, v := range s.state.Idempotency {
		next.Idempotency[k] = v
	}
	return next
}
func (s *Store) persist(state diskState) error {
	parent := filepath.Dir(s.path)
	if err := os.MkdirAll(parent, 0750); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	temp, err := os.CreateTemp(parent, ".issues-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary data: %w", err)
	}
	name := temp.Name()
	defer os.Remove(name)
	defer temp.Close()
	enc := json.NewEncoder(temp)
	if err = enc.Encode(state); err != nil {
		return fmt.Errorf("encode data: %w", err)
	}
	if err = temp.Sync(); err != nil {
		return fmt.Errorf("sync data: %w", err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	if err = os.Rename(name, s.path); err != nil {
		return fmt.Errorf("replace data: %w", err)
	}
	// Rename is the commit point. Directory sync is best-effort: reporting a failure
	// after this point would falsely tell a client that its committed write failed.
	if dir, err := os.Open(parent); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}
func validKey(key string) bool {
	if len(key) != 36 {
		return false
	}
	for _, index := range []int{8, 13, 18, 23} {
		if key[index] != '-' {
			return false
		}
	}
	raw := strings.ReplaceAll(key, "-", "")
	if len(raw) != 32 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}
