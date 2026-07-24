package chunk002

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Constants and versioning
// ---------------------------------------------------------------------------

const (
	// AuditLogVersion_014 is the version identifier for audit log entries.
	AuditLogVersion_014 = "v1.0"

	// DefaultRedactedPlaceholder is used to replace secret values.
	DefaultRedactedPlaceholder_014 = "[REDACTED]"
)

// ---------------------------------------------------------------------------
// Types for audit log entries
// ---------------------------------------------------------------------------

// ResourceAgentAuditLogEntry_014 represents a single auditable action.
type ResourceAgentAuditLogEntry_014 struct {
	Version       string            `json:"version"`
	Time          time.Time         `json:"time"`
	SessionID     string            `json:"session_id"`
	User          string            `json:"user"`
	Action        string            `json:"action"` // e.g. "CREATE", "DELETE", "EXEC"
	ResourceType  string            `json:"resource_type"`
	ResourceName  string            `json:"resource_name"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Environment   map[string]string `json:"environment"`
	Outcome       string            `json:"outcome"` // "SUCCESS", "FAILURE"
	ErrorMessage  string            `json:"error_message,omitempty"`
	SecretsHashes []string          `json:"secrets_hashes,omitempty"`
	RedactedArgs  []string          `json:"redacted_args,omitempty"`
}

// ResourceAgentAuditLog_014 is a collection of audit entries.
type ResourceAgentAuditLog_014 []ResourceAgentAuditLogEntry_014

// ---------------------------------------------------------------------------
// Redaction pattern definitions
// ---------------------------------------------------------------------------

// ResourceAgentRedactionPattern_014 holds a compiled regular expression and its replacement.
type ResourceAgentRedactionPattern_014 struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// redactionPatterns is a deterministic table of patterns used to redact secrets.
var redactionPatterns_014 = []ResourceAgentRedactionPattern_014{
	{Pattern: regexp.MustCompile(`(?i)(Bearer\s+)[a-zA-Z0-9\-._~+/]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(password\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(token\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(secret\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(access[_-]?key\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(secret[_-]?key\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(x-auth-token\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
	{Pattern: regexp.MustCompile(`(?i)(client_secret\s*[:=]\s*)[^\s&]+`), Replacement: "${1}" + DefaultRedactedPlaceholder_014},
}

// ---------------------------------------------------------------------------
// Redactor
// ---------------------------------------------------------------------------

// ResourceAgentRedactor_014 applies redaction patterns to strings and maps.
type ResourceAgentRedactor_014 struct {
	patterns []ResourceAgentRedactionPattern_014
	// Additional custom patterns can be added via AddPattern.
}

// NewResourceAgentRedactor_014 creates a redactor with the default secret patterns.
func NewResourceAgentRedactor_014() *ResourceAgentRedactor_014 {
	r := &ResourceAgentRedactor_014{
		patterns: make([]ResourceAgentRedactionPattern_014, len(redactionPatterns_014)),
	}
	copy(r.patterns, redactionPatterns_014)
	return r
}

// AddPattern adds a user-defined redaction pattern.
func (r *ResourceAgentRedactor_014) AddPattern(pattern *regexp.Regexp, replacement string) {
	r.patterns = append(r.patterns, ResourceAgentRedactionPattern_014{
		Pattern:     pattern,
		Replacement: replacement,
	})
}

// RedactString_014 applies all patterns to the input and returns the redacted string.
func (r *ResourceAgentRedactor_014) RedactString_014(input string) string {
	result := input
	for _, p := range r.patterns {
		result = p.Pattern.ReplaceAllString(result, p.Replacement)
	}
	return result
}

// RedactArgs_014 applies redaction to a slice of command arguments.
func (r *ResourceAgentRedactor_014) RedactArgs_014(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = r.RedactString_014(arg)
	}
	return out
}

// RedactEnvMap_014 applies redaction to environment variable key-value pairs.
func (r *ResourceAgentRedactor_014) RedactEnvMap_014(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = r.RedactString_014(v)
		// Also redact sensitive keys if desired (e.g., key names containing "SECRET")
	}
	return out
}

// ---------------------------------------------------------------------------
// Hashing
// ---------------------------------------------------------------------------

// ResourceAgentHasher_014 computes a hash for an audit log entry.
// The hash is deterministic and can be used for integrity verification.
type ResourceAgentHasher_014 struct {
	// ExcludeFields lists fields that should be omitted from hashing.
	ExcludeFields []string
}

// NewResourceAgentHasher_014 creates a hasher with sensible defaults.
func NewResourceAgentHasher_014() *ResourceAgentHasher_014 {
	return &ResourceAgentHasher_014{
		ExcludeFields: []string{"SecretsHashes", "RedactedArgs"},
	}
}

// ComputeEntryHash_014 returns the hex-encoded SHA‑256 hash of a canonical representation
// of the entry, excluding fields specified in ExcludeFields.
func (h *ResourceAgentHasher_014) ComputeEntryHash_014(entry *ResourceAgentAuditLogEntry_014) (string, error) {
	if entry == nil {
		return "", fmt.Errorf("cannot hash nil entry")
	}
	// Build canonical representation as a sorted map.
	canon := make(map[string]interface{})
	canon["version"] = entry.Version
	canon["time"] = entry.Time.UTC().Format(time.RFC3339Nano)
	canon["session_id"] = entry.SessionID
	canon["user"] = entry.User
	canon["action"] = entry.Action
	canon["resource_type"] = entry.ResourceType
	canon["resource_name"] = entry.ResourceName
	canon["command"] = entry.Command
	canon["args"] = entry.Args
	canon["environment"] = entry.Environment
	canon["outcome"] = entry.Outcome
	canon["error_message"] = entry.ErrorMessage

	// Remove excluded fields.
	for _, f := range h.ExcludeFields {
		delete(canon, f)
	}

	// Serialize to deterministic string.
	serialized := canonicalString_014(canon)
	hash := sha256.Sum256([]byte(serialized))
	return hex.EncodeToString(hash[:]), nil
}

// canonicalString_014 converts a map to a deterministic string representation.
// Keys are sorted, values are printed using %v with newline separators.
func canonicalString_014(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(fmt.Sprintf("%v", m[k]))
		b.WriteString("\n")
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// ValidateAuditLogEntry_014 checks that the entry contains all required fields
// and returns an error if any constraint is violated.
func ValidateAuditLogEntry_014(entry *ResourceAgentAuditLogEntry_014) error {
	if entry == nil {
		return fmt.Errorf("entry is nil")
	}
	if entry.Time.IsZero() {
		return fmt.Errorf("time field is zero")
	}
	if entry.SessionID == "" {
		return fmt.Errorf("session_id is empty")
	}
	if entry.User == "" {
		return fmt.Errorf("user is empty")
	}
	validActions := map[string]bool{
		"CREATE": true, "READ": true, "UPDATE": true, "DELETE": true,
		"EXEC": true, "MODIFY": true, "LIST": true, "AUTH": true,
	}
	if !validActions[entry.Action] {
		return fmt.Errorf("unknown action: %q", entry.Action)
	}
	if entry.ResourceType == "" {
		return fmt.Errorf("resource_type is empty")
	}
	if entry.ResourceName == "" {
		return fmt.Errorf("resource_name is empty")
	}
	if entry.Command == "" {
		return fmt.Errorf("command is empty")
	}
	if entry.Outcome != "SUCCESS" && entry.Outcome != "FAILURE" {
		return fmt.Errorf("outcome must be SUCCESS or FAILURE, got %q", entry.Outcome)
	}
	if entry.Outcome == "FAILURE" && entry.ErrorMessage == "" {
		return fmt.Errorf("error_message is required when outcome is FAILURE")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

// ResourceAgentNewEntry_014 creates a basic audit entry with version and current timestamp.
func ResourceAgentNewEntry_014(sessionID, user, action, resourceType, resourceName, command string, args []string, env map[string]string) *ResourceAgentAuditLogEntry_014 {
	return &ResourceAgentAuditLogEntry_014{
		Version:      AuditLogVersion_014,
		Time:         time.Now().UTC(),
		SessionID:    sessionID,
		User:         user,
		Action:       action,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Command:      command,
		Args:         args,
		Environment:  env,
		Outcome:      "SUCCESS",
	}
}

// ResourceAgentRedactAndHashEntry_014 applies redaction and hashing to an entry.
// The redactor is used to produce redacted copies of Args and Environment.
// The hasher computes a hash of the original (unredacted) entry and stores it in SecretsHashes.
func ResourceAgentRedactAndHashEntry_014(entry *ResourceAgentAuditLogEntry_014, redactor *ResourceAgentRedactor_014, hasher *ResourceAgentHasher_014) (*ResourceAgentAuditLogEntry_014, error) {
	if entry == nil {
		return nil, fmt.Errorf("entry is nil")
	}
	if redactor == nil {
		return nil, fmt.Errorf("redactor is nil")
	}
	if hasher == nil {
		return nil, fmt.Errorf("hasher is nil")
	}

	// Redact arguments and environment.
	redactedArgs := redactor.RedactArgs_014(entry.Args)
	redactedEnv := redactor.RedactEnvMap_014(entry.Environment)

	// Compute hash of original entry (excluding redacted fields).
	hash, err := hasher.ComputeEntryHash_014(entry)
	if err != nil {
		return nil, fmt.Errorf("computing hash: %w", err)
	}

	// Build the audit entry that will be stored.
	auditEntry := *entry // copy
	auditEntry.Args = redactedArgs
	auditEntry.Environment = redactedEnv
	auditEntry.SecretsHashes = []string{hash}
	auditEntry.RedactedArgs = redactedArgs // also store for clarity
	return &auditEntry, nil
}

// ResourceAgentAuditLogBatch_014 represents a batch of audit entries.
type ResourceAgentAuditLogBatch_014 struct {
	Entries  []ResourceAgentAuditLogEntry_014 `json:"entries"`
	BatchID  string                           `json:"batch_id"`
	Created  time.Time                        `json:"created"`
}

// ResourceAgentNewBatch_014 creates a new batch with a deterministic BatchID based on hash of entries.
func ResourceAgentNewBatch_014(entries []ResourceAgentAuditLogEntry_014) *ResourceAgentAuditLogBatch_014 {
	batch := &ResourceAgentAuditLogBatch_014{
		Entries: entries,
		Created: time.Now().UTC(),
	}
	// Compute a simple hash for batch identification.
	var b strings.Builder
	for _, e := range entries {
		h, _ := NewResourceAgentHasher_014().ComputeEntryHash_014(&e)
		b.WriteString(h)
	}
	sum := sha256.Sum256([]byte(b.String()))
	batch.BatchID = hex.EncodeToString(sum[:8]) // first 8 bytes as hex
	return batch
}

// ---------------------------------------------------------------------------
// Deterministic helper tables (not random padding)
// ---------------------------------------------------------------------------

// ResourceAgentSensitiveEnvKeys_014 returns a sorted list of environment variable keys
// that typically contain secret values. This is used by redaction to optionally
// treat entire values as secrets.
func ResourceAgentSensitiveEnvKeys_014() []string {
	return []string{
		"ACCESS_KEY_ID",
		"API_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"CLIENT_SECRET",
		"DB_PASSWORD",
		"GITHUB_TOKEN",
		"PASSWORD",
		"SECRET",
		"SECRET_KEY",
		"TOKEN",
	}
}

// ResourceAgentAllowedActions_014 returns the set of allowed actions for validation.
func ResourceAgentAllowedActions_014() map[string]bool {
	return map[string]bool{
		"CREATE": true, "READ": true, "UPDATE": true, "DELETE": true,
		"EXEC":   true, "MODIFY": true, "LIST": true, "AUTH": true,
	}
}

// ResourceAgentPatternTable_014 returns a human-readable representation of the internal redaction patterns.
func ResourceAgentPatternTable_014() []string {
	table := make([]string, 0, len(redactionPatterns_014))
	for _, p := range redactionPatterns_014 {
		table = append(table, fmt.Sprintf("%s -> %s", p.Pattern.String(), p.Replacement))
	}
	return table
}

// ResourceAgentFieldOrder_014 defines the canonical order of fields for hashing.
func ResourceAgentFieldOrder_014() []string {
	return []string{
		"version", "time", "session_id", "user", "action",
		"resource_type", "resource_name", "command", "args",
		"environment", "outcome", "error_message",
	}
}

// ---------------------------------------------------------------------------
// Additional validation helper
// ---------------------------------------------------------------------------

// ValidateAuditLogBatch_014 validates all entries in a batch.
func ValidateAuditLogBatch_014(batch *ResourceAgentAuditLogBatch_014) error {
	if batch == nil {
		return fmt.Errorf("batch is nil")
	}
	if len(batch.Entries) == 0 {
		return fmt.Errorf("batch contains no entries")
	}
	if batch.Created.IsZero() {
		return fmt.Errorf("batch created time is zero")
	}
	for i, entry := range batch.Entries {
		if err := ValidateAuditLogEntry_014(&entry); err != nil {
			return fmt.Errorf("entry %d: %w", i, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// End of file
// ---------------------------------------------------------------------------
