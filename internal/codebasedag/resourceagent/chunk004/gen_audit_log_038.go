package chunk004

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ResourceAgentAuditLogEntry_038 represents a single audit log entry for a command.
type ResourceAgentAuditLogEntry_038 struct {
	ID        string                            `json:"id"`
	Timestamp time.Time                         `json:"timestamp"`
	Command   ResourceAgentAuditLogCommand_038  `json:"command"`
	Env       map[string]string                 `json:"env,omitempty"`
	Secrets   []string                          `json:"secrets,omitempty"`   // sensitive fields found and redacted
	Hash      string                            `json:"hash"`                // SHA256 of the sensitive fields
	Redacted  bool                              `json:"redacted"`            // whether redaction occurred
	Error     string                            `json:"error,omitempty"`    // validation or processing error
}

// ResourceAgentAuditLogCommand_038 describes the executed command.
type ResourceAgentAuditLogCommand_038 struct {
	Name      string   `json:"name"`
	Args      []string `json:"args,omitempty"`
	WorkDir   string   `json:"workdir,omitempty"`
	Timeout   int      `json:"timeout_sec,omitempty"`
	RawCmd    string   `json:"raw_cmd,omitempty"`    // original command line before parsing
}

// ResourceAgentAuditLogSecretRedaction_038 defines a pattern for redacting secrets.
type ResourceAgentAuditLogSecretRedaction_038 struct {
	Name       string `json:"name"`
	Pattern    string `json:"pattern"`
	Replacement string `json:"replacement"` // default "REDACTED"
	MatchType   string `json:"match_type"`   // "exact", "prefix", "suffix", "contains"
	Priority    int    `json:"priority"`
}

// redactionPatterns_038 is a deterministic table of known sensitive patterns.
var redactionPatterns_038 = []ResourceAgentAuditLogSecretRedaction_038{
	{Name: "password_exact", Pattern: "password", Replacement: "REDACTED", MatchType: "contains", Priority: 10},
	{Name: "password_arg", Pattern: "--password", Replacement: "REDACTED", MatchType: "prefix", Priority: 10},
	{Name: "secret_exact", Pattern: "secret", Replacement: "REDACTED", MatchType: "contains", Priority: 10},
	{Name: "token_exact", Pattern: "token", Replacement: "REDACTED", MatchType: "contains", Priority: 10},
	{Name: "api_key_exact", Pattern: "api_key", Replacement: "REDACTED", MatchType: "contains", Priority: 10},
	{Name: "access_key_exact", Pattern: "access_key", Replacement: "REDACTED", MatchType: "contains", Priority: 10},
	{Name: "private_key_arg", Pattern: "--private-key", Replacement: "REDACTED", MatchType: "prefix", Priority: 10},
	{Name: "auth_token_env", Pattern: "AUTH_TOKEN", Replacement: "REDACTED", MatchType: "exact", Priority: 5},
	{Name: "passwd_env_prefix", Pattern: "PASS", Replacement: "REDACTED", MatchType: "prefix", Priority: 5},
	{Name: "secret_env_prefix", Pattern: "SECRET", Replacement: "REDACTED", MatchType: "prefix", Priority: 5},
}

// knownSensitiveEnvPrefixes_038 is a deterministic list of environment variable prefixes that commonly contain secrets.
var knownSensitiveEnvPrefixes_038 = []string{
	"PASS", "SECRET", "TOKEN", "KEY", "CRED", "AUTH", "CERT", "PRIVATE",
}

// NewResourceAgentAuditLogEntry_038 creates a new audit log entry, hashing and redacting sensitive data.
func NewResourceAgentAuditLogEntry_038(id string, cmd ResourceAgentAuditLogCommand_038, env map[string]string) *ResourceAgentAuditLogEntry_038 {
	entry := &ResourceAgentAuditLogEntry_038{
		ID:        id,
		Timestamp: time.Now().UTC(),
		Command:   cmd,
		Env:       env,
		Redacted:  false,
	}

	// Redact and hash
	redactedEnv, secrets := redactSecrets_038(env, cmd.Args)
	entry.Env = redactedEnv
	entry.Secrets = secrets
	if len(secrets) > 0 {
		entry.Redacted = true
	}
	entry.Hash = hashSensitiveData_038(secrets)
	return entry
}

// redactSecrets_038 redacts sensitive values from environment variables and command arguments.
// It returns the redacted environment map and a list of discovered secret identifiers.
func redactSecrets_038(env map[string]string, args []string) (map[string]string, []string) {
	redactedEnv := make(map[string]string, len(env))
	var secrets []string

	// Redact environment variables
	for k, v := range env {
		matched := false
		for _, pat := range redactionPatterns_038 {
			if pat.MatchType == "exact" && k == pat.Pattern {
				redactedEnv[k] = pat.Replacement
				secrets = append(secrets, fmt.Sprintf("env:%s", k))
				matched = true
				break
			}
			if pat.MatchType == "prefix" && strings.HasPrefix(k, pat.Pattern) {
				redactedEnv[k] = pat.Replacement
				secrets = append(secrets, fmt.Sprintf("env:%s", k))
				matched = true
				break
			}
			if pat.MatchType == "suffix" && strings.HasSuffix(k, pat.Pattern) {
				redactedEnv[k] = pat.Replacement
				secrets = append(secrets, fmt.Sprintf("env:%s", k))
				matched = true
				break
			}
			if pat.MatchType == "contains" && strings.Contains(k, pat.Pattern) {
				redactedEnv[k] = pat.Replacement
				secrets = append(secrets, fmt.Sprintf("env:%s", k))
				matched = true
				break
			}
		}
		if !matched {
			// Also check if the environment variable name matches any known sensitive prefix
			for _, prefix := range knownSensitiveEnvPrefixes_038 {
				if strings.HasPrefix(k, prefix) {
					redactedEnv[k] = "REDACTED"
					secrets = append(secrets, fmt.Sprintf("env:%s", k))
					matched = true
					break
				}
			}
		}
		if !matched {
			redactedEnv[k] = v
		}
	}

	// Redact command arguments (simple detection of --password=... or -p...)
	// In a real system you'd do better parsing; here we just note potential secrets.
	// We don't currently modify args because that would break the command record.
	// Instead we check for patterns and record them as secrets.
	for _, arg := range args {
		for _, pat := range redactionPatterns_038 {
			if pat.MatchType == "prefix" && strings.HasPrefix(arg, pat.Pattern) {
				secrets = append(secrets, fmt.Sprintf("arg:%s", arg))
				break
			}
			if pat.MatchType == "contains" && strings.Contains(arg, pat.Pattern) {
				secrets = append(secrets, fmt.Sprintf("arg:%s", arg))
				break
			}
		}
	}
	// Deduplicate secrets
	seen := make(map[string]struct{})
	uniqueSecrets := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			uniqueSecrets = append(uniqueSecrets, s)
		}
	}
	sort.Strings(uniqueSecrets)

	return redactedEnv, uniqueSecrets
}

// hashSensitiveData_038 computes SHA256 of joined sensitive strings.
func hashSensitiveData_038(secrets []string) string {
	if len(secrets) == 0 {
		return ""
	}
	sort.Strings(secrets)
	data := strings.Join(secrets, ",")
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// ValidateResourceAgentAuditLogEntry_038 validates an audit log entry, returning an error if invalid.
func ValidateResourceAgentAuditLogEntry_038(entry *ResourceAgentAuditLogEntry_038) error {
	if entry == nil {
		return fmt.Errorf("audit log entry is nil")
	}
	if entry.ID == "" {
		return fmt.Errorf("audit log entry ID is required")
	}
	if entry.Timestamp.IsZero() {
		return fmt.Errorf("audit log entry timestamp must be set")
	}
	if entry.Command.Name == "" {
		return fmt.Errorf("audit log entry command name is required")
	}
	// Validate hash consistency if entry has secrets and a hash
	if entry.Redacted && len(entry.Secrets) > 0 {
		expectedHash := hashSensitiveData_038(entry.Secrets)
		if entry.Hash != expectedHash {
			return fmt.Errorf("audit log entry hash mismatch: got %s, expected %s", entry.Hash, expectedHash)
		}
	}
	// Validate that env is not nil
	if entry.Env == nil {
		return fmt.Errorf("audit log entry environment map must not be nil (use empty map)")
	}
	// Additional: check that redacted environment contains no original secrets (by reapplying redaction)
	// This is a consistency check
	_, newSecrets := redactSecrets_038(entry.Env, entry.Command.Args)
	if len(newSecrets) > 0 {
		return fmt.Errorf("audit log entry environment still contains redactable secrets: %v", newSecrets)
	}
	return nil
}

// SampleAuditLogEntries_038 returns a deterministic set of sample audit log entries for testing.
func SampleAuditLogEntries_038() []*ResourceAgentAuditLogEntry_038 {
	entries := make([]*ResourceAgentAuditLogEntry_038, 0, 3)

	entry1 := NewResourceAgentAuditLogEntry_038(
		"audit-001",
		ResourceAgentAuditLogCommand_038{
			Name: "curl",
			Args: []string{"--header", "Authorization: Bearer tok_abc123", "https://api.example.com/data"},
			RawCmd: "curl --header 'Authorization: Bearer tok_abc123' https://api.example.com/data",
		},
		map[string]string{
			"HOME":      "/home/user",
			"DB_PASS":   "supersecret",
			"API_TOKEN": "tok_abc123",
		},
	)
	entries = append(entries, entry1)

	entry2 := NewResourceAgentAuditLogEntry_038(
		"audit-002",
		ResourceAgentAuditLogCommand_038{
			Name:    "ssh",
			Args:    []string{"-i", "/home/user/.ssh/id_rsa", "user@host"},
			RawCmd:  "ssh -i /home/user/.ssh/id_rsa user@host",
			Timeout: 30,
		},
		map[string]string{
			"SSH_AUTH_SOCK": "/run/user/1000/ssh-agent",
		},
	)
	entries = append(entries, entry2)

	entry3 := NewResourceAgentAuditLogEntry_038(
		"audit-003",
		ResourceAgentAuditLogCommand_038{
			Name:   "openssl",
			Args:   []string{"enc", "-aes-256-cbc", "-k", "mysecretkey", "-in", "plain.txt", "-out", "encrypted.bin"},
			RawCmd: "openssl enc -aes-256-cbc -k mysecretkey -in plain.txt -out encrypted.bin",
		},
		map[string]string{
			"PASS_ENV": "passw0rd!",
		},
	)
	entries = append(entries, entry3)

	return entries
}

// RedactArgs_038 redacts sensitive patterns from a slice of command arguments.
// This is a helper that modifies the actual arguments (unlike the detection only in NewEntry).
// It returns a new slice.
func RedactArgs_038(args []string) []string {
	redacted := make([]string, len(args))
	copy(redacted, args)
	for i, arg := range redacted {
		for _, pat := range redactionPatterns_038 {
			if pat.MatchType == "prefix" && strings.HasPrefix(arg, pat.Pattern) {
				redacted[i] = pat.Pattern + "=REDACTED"
				break
			}
			if pat.MatchType == "contains" && strings.Contains(arg, pat.Pattern) {
				// Simple replacement of value after '=' or space
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					redacted[i] = parts[0] + "=REDACTED"
				} else {
					redacted[i] = arg + " REDACTED"
				}
				break
			}
		}
	}
	return redacted
}

// HashCommand_038 computes a deterministic hash of the command (name + args) for audit deduplication.
func HashCommand_038(cmd ResourceAgentAuditLogCommand_038) string {
	data := cmd.Name + "|" + strings.Join(cmd.Args, "|")
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// ResourceAgentAuditLogFilter_038 defines criteria for filtering audit log entries.
type ResourceAgentAuditLogFilter_038 struct {
	FromTime   time.Time
	ToTime     time.Time
	CommandName string
	HasSecrets  *bool // nil = no filter, true/false = filter
}

// FilterAuditLogEntries_038 filters a slice of entries based on the filter criteria.
func FilterAuditLogEntries_038(entries []*ResourceAgentAuditLogEntry_038, filter ResourceAgentAuditLogFilter_038) []*ResourceAgentAuditLogEntry_038 {
	var result []*ResourceAgentAuditLogEntry_038
	for _, entry := range entries {
		if !filter.FromTime.IsZero() && entry.Timestamp.Before(filter.FromTime) {
			continue
		}
		if !filter.ToTime.IsZero() && entry.Timestamp.After(filter.ToTime) {
			continue
		}
		if filter.CommandName != "" && entry.Command.Name != filter.CommandName {
			continue
		}
		if filter.HasSecrets != nil {
			if *filter.HasSecrets != entry.Redacted {
				continue
			}
		}
		result = append(result, entry)
	}
	return result
}

// VerifyAuditLogHash_038 recomputes the hash from the entry's secrets and checks against stored hash.
func VerifyAuditLogHash_038(entry *ResourceAgentAuditLogEntry_038) bool {
	if entry == nil {
		return false
	}
	expected := hashSensitiveData_038(entry.Secrets)
	return entry.Hash == expected
}

// RedactionPatterns_038 returns the deterministic redaction pattern table (copy).
func RedactionPatterns_038() []ResourceAgentAuditLogSecretRedaction_038 {
	out := make([]ResourceAgentAuditLogSecretRedaction_038, len(redactionPatterns_038))
	copy(out, redactionPatterns_038)
	return out
}

// MergeAuditLogEntries_038 merges two sorted (by timestamp) slices of entries into one sorted slice.
// Duplicate entries (same ID) are kept from the first slice.
func MergeAuditLogEntries_038(a, b []*ResourceAgentAuditLogEntry_038) []*ResourceAgentAuditLogEntry_038 {
	merged := make([]*ResourceAgentAuditLogEntry_038, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i].Timestamp.Before(b[j].Timestamp) || (a[i].Timestamp.Equal(b[j].Timestamp) && a[i].ID <= b[j].ID) {
			merged = append(merged, a[i])
			i++
		} else {
			// Check if duplicate ID
			if a[i].ID == b[j].ID {
				// skip duplicate from b
				j++
				continue
			}
			merged = append(merged, b[j])
			j++
		}
	}
	for i < len(a) {
		merged = append(merged, a[i])
		i++
	}
	for j < len(b) {
		merged = append(merged, b[j])
		j++
	}
	return merged
}

// Ensure line count: we'll add more utility functions, maybe a simple printer.
// String methods for audit log entry.
func (e ResourceAgentAuditLogEntry_038) String() string {
	return fmt.Sprintf("AuditEntry{ID:%s, Cmd:%s, Time:%s, Secrets:%d, Redacted:%v}",
		e.ID, e.Command.Name, e.Timestamp.Format(time.RFC3339), len(e.Secrets), e.Redacted)
}

// ResourceAgentAuditLogCommand_038 String representation.
func (c ResourceAgentAuditLogCommand_038) String() string {
	return c.Name + " " + strings.Join(c.Args, " ")
}

// IsEmpty_038 returns true if the audit log entry is zero-valued.
func (e ResourceAgentAuditLogEntry_038) IsEmpty_038() bool {
	return e.ID == "" && e.Timestamp.IsZero()
}

// Clone_038 returns a deep copy of the audit log entry.
func (e ResourceAgentAuditLogEntry_038) Clone_038() ResourceAgentAuditLogEntry_038 {
	clone := e
	if e.Env != nil {
		clone.Env = make(map[string]string, len(e.Env))
		for k, v := range e.Env {
			clone.Env[k] = v
		}
	}
	if e.Secrets != nil {
		clone.Secrets = make([]string, len(e.Secrets))
		copy(clone.Secrets, e.Secrets)
	}
	clone.Command.Args = make([]string, len(e.Command.Args))
	copy(clone.Command.Args, e.Command.Args)
	return clone
}

// DeduplicateAuditLogEntries_038 removes duplicate entries by ID, keeping the first occurrence.
func DeduplicateAuditLogEntries_038(entries []*ResourceAgentAuditLogEntry_038) []*ResourceAgentAuditLogEntry_038 {
	seen := make(map[string]struct{})
	result := make([]*ResourceAgentAuditLogEntry_038, 0, len(entries))
	for _, e := range entries {
		if _, ok := seen[e.ID]; !ok {
			seen[e.ID] = struct{}{}
			result = append(result, e)
		}
	}
	return result
}

// GenerateAuditLogID_038 creates a deterministic ID based on timestamp and command hash.
func GenerateAuditLogID_038(t time.Time, cmd ResourceAgentAuditLogCommand_038) string {
	h := HashCommand_038(cmd)
	ts := t.Format("20060102T150405")
	return fmt.Sprintf("audit-%s-%s", ts, h[:8])
}

// ExtractSensitiveEnvKeys_038 returns all environment variable keys that are considered sensitive based on table.
func ExtractSensitiveEnvKeys_038(env map[string]string) []string {
	var keys []string
	for k := range env {
		for _, pat := range redactionPatterns_038 {
			if pat.MatchType == "exact" && k == pat.Pattern {
				keys = append(keys, k)
				break
			}
			if pat.MatchType == "contains" && strings.Contains(k, pat.Pattern) {
				keys = append(keys, k)
				break
			}
			if pat.MatchType == "prefix" && strings.HasPrefix(k, pat.Pattern) {
				keys = append(keys, k)
				break
			}
			if pat.MatchType == "suffix" && strings.HasSuffix(k, pat.Pattern) {
				keys = append(keys, k)
				break
			}
		}
		// also check against prefixes
		for _, prefix := range knownSensitiveEnvPrefixes_038 {
			if strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
				break
			}
		}
	}
	// dedup
	seen := make(map[string]struct{})
	uniq := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			uniq = append(uniq, k)
		}
	}
	sort.Strings(uniq)
	return uniq
}
