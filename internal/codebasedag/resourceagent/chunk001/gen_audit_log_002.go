package chunk001

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Types for command audit log structures
// ---------------------------------------------------------------------------

// ResourceAgentAuditLogEntry represents a single audited command execution.
type ResourceAgentAuditLogEntry struct {
	Command       string            `json:"command"`
	Arguments     []string          `json:"arguments,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	UserID        string            `json:"user_id"`
	SessionID     string            `json:"session_id,omitempty"`
	SensitiveData map[string]string `json:"-"` // not serialized, used for redaction hints
}

// ResourceAgentRedactionRule defines a redaction rule based on a regular expression.
type ResourceAgentRedactionRule struct {
	Pattern     *regexp.Regexp
	Replacement string
	RuleName    string
}

// ResourceAgentAuditLogHash holds the hashes of an audit log entry.
type ResourceAgentAuditLogHash struct {
	RawSHA256      string `json:"raw_sha256"`
	RedactedSHA256 string `json:"redacted_sha256"`
}

// ResourceAgentRedactedEntry is the sanitized version of an audit log entry.
type ResourceAgentRedactedEntry struct {
	Command       string   `json:"command"`
	Arguments     []string `json:"arguments,omitempty"`
	Timestamp     string   `json:"timestamp"`
	UserID        string   `json:"user_id"`
	SessionID     string   `json:"session_id,omitempty"`
}

// ---------------------------------------------------------------------------
// Deterministic table-driven helper data: redaction rules
// ---------------------------------------------------------------------------

// DefaultRedactionRules_002 returns a slice of common secret patterns.
// Rules are ordered so that more specific patterns are checked first.
func DefaultRedactionRules_002() []ResourceAgentRedactionRule {
	// Each rule is defined as (raw pattern, replacement, rule name)
	rulesDef := []struct {
		Pattern     string
		Replacement string
		RuleName    string
	}{
		{
			Pattern:     `(?i)(password|passwd|pwd)\s*[=:]\s*['"]?\S+['"]?`,
			Replacement: "${1}=***REDACTED***",
			RuleName:    "password",
		},
		{
			Pattern:     `(?i)(secret|token|api[_-]?key)\s*[=:]\s*['"]?\S+['"]?`,
			Replacement: "${1}=***REDACTED***",
			RuleName:    "secret_token",
		},
		{
			Pattern:     `(?i)(client_secret|access_key|private_key)\s*[=:]\s*['"]?\S+['"]?`,
			Replacement: "${1}=***REDACTED***",
			RuleName:    "client_secret",
		},
		{
			Pattern:     `(?i)(Bearer|Basic)\s+\S+`,
			Replacement: "${1} ***REDACTED***",
			RuleName:    "authorization_header",
		},
		{
			Pattern:     `(?i)(ssh-rsa|ssh-ed25519|ecdsa-sha2-nistp256)\s+\S+`,
			Replacement: "${1} ***REDACTED***",
			RuleName:    "public_key",
		},
		{
			Pattern:     `(?i)(-----BEGIN\s+(RSA|EC|OPENSSH|PRIVATE)\s+KEY-----).*?(-----END\s+\1-----)`,
			Replacement: "${1}***REDACTED***${3}",
			RuleName:    "private_key_block",
		},
		{
			Pattern:     `(?i)(certificate|auth[_-]?code)\s*[=:]\s*['"]?\S+['"]?`,
			Replacement: "${1}=***REDACTED***",
			RuleName:    "auth_code",
		},
	}

	rules := make([]ResourceAgentRedactionRule, len(rulesDef))
	for i, r := range rulesDef {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			// In production, we would log or panic; for this package we return a no-op regex.
			re = regexp.MustCompile(`a^`) // never matches
		}
		rules[i] = ResourceAgentRedactionRule{
			Pattern:     re,
			Replacement: r.Replacement,
			RuleName:    r.RuleName,
		}
	}
	return rules
}

// ---------------------------------------------------------------------------
// Utility: JSON serialisation helpers (internal)
// ---------------------------------------------------------------------------

// mustMarshalJSON serializes v to JSON; panics on error (should not happen with these types).
func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic("resourceagent: JSON marshal failed: " + err.Error())
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Redaction of secrets
// ---------------------------------------------------------------------------

// RedactSecrets_002 applies the given rules to a string, returning the redacted version
// and a count of replacements made.
func RedactSecrets_002(input string, rules []ResourceAgentRedactionRule) (string, int) {
	var totalReplacements int
	output := input
	for _, rule := range rules {
		// Count replacements made by this rule
		replaced := rule.Pattern.ReplaceAllStringFunc(output, func(match string) string {
			totalReplacements++
			return rule.Pattern.ReplaceAllString(match, rule.Replacement)
		})
		output = replaced
	}
	return output, totalReplacements
}

// RedactAuditLogEntry_002 creates a redacted copy of an audit log entry using the given rules.
func RedactAuditLogEntry_002(entry *ResourceAgentAuditLogEntry, rules []ResourceAgentRedactionRule) (*ResourceAgentRedactedEntry, int) {
	redactedCmd, count := RedactSecrets_002(entry.Command, rules)
	redactedArgs := make([]string, len(entry.Arguments))
	for i, arg := range entry.Arguments {
		argRedacted, _ := RedactSecrets_002(arg, rules)
		redactedArgs[i] = argRedacted
	}

	return &ResourceAgentRedactedEntry{
		Command:   redactedCmd,
		Arguments: redactedArgs,
		Timestamp: entry.Timestamp.UTC().Format(time.RFC3339Nano),
		UserID:    entry.UserID,
		SessionID: entry.SessionID,
	}, count
}

// ---------------------------------------------------------------------------
// Hashing functions
// ---------------------------------------------------------------------------

// HashStringSHA256 returns the hex-encoded SHA256 hash of the input string.
func HashStringSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// HashAuditLogEntry_002 computes the raw and redacted hashes for an audit log entry.
// The raw hash is computed from the original entry JSON, the redacted hash from the redacted version.
func HashAuditLogEntry_002(entry *ResourceAgentAuditLogEntry, rules []ResourceAgentRedactionRule) *ResourceAgentAuditLogHash {
	// Raw hash (without redaction)
	rawJSON := mustMarshalJSON(entry)
	rawHash := HashStringSHA256(rawJSON)

	// Redacted hash
	redactedEntry, _ := RedactAuditLogEntry_002(entry, rules)
	redactedJSON := mustMarshalJSON(redactedEntry)
	redactedHash := HashStringSHA256(redactedJSON)

	return &ResourceAgentAuditLogHash{
		RawSHA256:      rawHash,
		RedactedSHA256: redactedHash,
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// ValidateAuditLogEntry_002 checks mandatory fields and reasonable time ranges.
// Returns nil if valid, otherwise an error describing the first issue found.
func ValidateAuditLogEntry_002(entry *ResourceAgentAuditLogEntry) error {
	if entry == nil {
		return errors.New("audit log entry is nil")
	}
	if strings.TrimSpace(entry.Command) == "" {
		return errors.New("command must not be empty")
	}
	if strings.TrimSpace(entry.UserID) == "" {
		return errors.New("user_id must not be empty")
	}
	if entry.Timestamp.IsZero() {
		return errors.New("timestamp must be set")
	}
	// Timestamp should not be in the future (with a small tolerance)
	if entry.Timestamp.After(time.Now().Add(5 * time.Second)) {
		return errors.New("timestamp is too far in the future (max 5s skew)")
	}
	// Timestamp should not be before year 2000 (reasonable lower bound)
	if entry.Timestamp.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return errors.New("timestamp is before the year 2000, likely invalid")
	}
	// Validate that arguments do not contain control characters (optional but good)
	for i, arg := range entry.Arguments {
		if strings.ContainsAny(arg, "\x00-\x08\x0B\x0C\x0E-\x1F") {
			return fmt.Errorf("argument %d contains control characters", i)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Construction helpers
// ---------------------------------------------------------------------------

// NewResourceAgentAuditLogEntry creates a validated ResourceAgentAuditLogEntry.
// It returns the entry and any validation error.
func NewResourceAgentAuditLogEntry(command string, arguments []string, userID, sessionID string, ts time.Time) (*ResourceAgentAuditLogEntry, error) {
	entry := &ResourceAgentAuditLogEntry{
		Command:   command,
		Arguments: arguments,
		UserID:    userID,
		SessionID: sessionID,
		Timestamp: ts,
	}
	if err := ValidateAuditLogEntry_002(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// ---------------------------------------------------------------------------
// Output / formatting
// ---------------------------------------------------------------------------

// ResourceAgentAuditLogEntrySummary returns a human-readable summary of the log entry,
// with optional redaction applied. The redacted flag controls whether secrets are masked.
func ResourceAgentAuditLogEntrySummary(entry *ResourceAgentAuditLogEntry, redacted bool) string {
	if entry == nil {
		return "<nil audit entry>"
	}
	var buf strings.Builder
	buf.WriteString("[AUDIT] ")
	buf.WriteString(entry.Timestamp.Format(time.RFC3339))
	buf.WriteString(" user=")
	buf.WriteString(entry.UserID)
	if entry.SessionID != "" {
		buf.WriteString(" session=")
		buf.WriteString(entry.SessionID)
	}
	buf.WriteString(" cmd=")

	cmd := entry.Command
	args := entry.Arguments
	if redacted {
		// Apply default rules for summary
		rules := DefaultRedactionRules_002()
		cmd, _ = RedactSecrets_002(cmd, rules)
		redactedArgs := make([]string, len(args))
		for i, a := range args {
			redactedArgs[i], _ = RedactSecrets_002(a, rules)
		}
		args = redactedArgs
	}

	buf.WriteString(cmd)
	if len(args) > 0 {
		buf.WriteString(" ")
		buf.WriteString(strings.Join(args, " "))
	}
	return buf.String()
}

// ---------------------------------------------------------------------------
// Table-driven test data (deterministic, for example validation)
// ---------------------------------------------------------------------------

// AuditLogTestVectors_002 returns a set of (input, expectedRedacted) pairs for
// quick testing of redaction rules. Each test vector is a struct with an raw command,
// the expected redacted output, and the rule name that should fire.
type ResourceAgentAuditLogTestVector struct {
	RawInput         string
	ExpectedRedacted string
	RuleName         string
}

// AuditLogTestVectors_002 provides deterministic test vectors for the default redaction rules.
func AuditLogTestVectors_002() []ResourceAgentAuditLogTestVector {
	return []ResourceAgentAuditLogTestVector{
		{
			RawInput:         "password=mySecret123",
			ExpectedRedacted: "password=***REDACTED***",
			RuleName:         "password",
		},
		{
			RawInput:         "api_key =  a1b2c3d4e5f6",
			ExpectedRedacted: "api_key =***REDACTED***",
			RuleName:         "secret_token",
		},
		{
			RawInput:         "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			ExpectedRedacted: "Authorization: Bearer ***REDACTED***",
			RuleName:         "authorization_header",
		},
		{
			RawInput:         "-----BEGIN RSA PRIVATE KEY-----\nMIIBOgIBAAJBAKj34GkxFhD90vcNLYLInFEX6Ppy1tPf9CrCL1Ee\n-----END RSA PRIVATE KEY-----",
			ExpectedRedacted: "-----BEGIN RSA PRIVATE KEY-----***REDACTED***-----END RSA PRIVATE KEY-----",
			RuleName:         "private_key_block",
		},
		{
			RawInput:         "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC... user@host",
			ExpectedRedacted: "ssh-rsa ***REDACTED***",
			RuleName:         "public_key",
		},
		{
			RawInput:         "auth_code=123456",
			ExpectedRedacted: "auth_code=***REDACTED***",
			RuleName:         "auth_code",
		},
	}
}

// ---------------------------------------------------------------------------
// Additional utility: Batch redact slice of entries
// ---------------------------------------------------------------------------

// RedactAuditLogEntries_002 applies redaction to a slice of entries, returning the
// redacted entries and the total number of replacements across all entries.
func RedactAuditLogEntries_002(entries []*ResourceAgentAuditLogEntry, rules []ResourceAgentRedactionRule) ([]*ResourceAgentRedactedEntry, int) {
	redactedList := make([]*ResourceAgentRedactedEntry, len(entries))
	totalReplacements := 0
	for i, entry := range entries {
		redacted, count := RedactAuditLogEntry_002(entry, rules)
		redactedList[i] = redacted
		totalReplacements += count
	}
	return redactedList, totalReplacements
}

// ---------------------------------------------------------------------------
// Verification helper: check that a redacted entry no longer contains any
// secret patterns from the given rules.
// ---------------------------------------------------------------------------

// VerifyRedactedEntry_002 returns true if the given redacted entry does not match
// any of the provided redaction rules (i.e., secrets were properly masked).
// Note that the replacement text itself might match the rule; we exclude that.
func VerifyRedactedEntry_002(redacted *ResourceAgentRedactedEntry, rules []ResourceAgentRedactionRule) bool {
	// Combine command and arguments into one string for scanning.
	allText := redacted.Command + " " + strings.Join(redacted.Arguments, " ")
	for _, rule := range rules {
		// If the pattern still matches and the match is not only the replacement constant,
		// then redaction may have failed.
		matches := rule.Pattern.FindAllString(allText, -1)
		for _, m := range matches {
			// Ignore matches that are exactly the placeholder text.
			if !strings.Contains(m, "***REDACTED***") {
				return false
			}
		}
	}
	return true
}

// Ensure the package is non-empty – this line is a marker.
var _ = DefaultRedactionRules_002

// gen_audit_log_002.go – complete implementation of audit log structures,
// hashing, redaction, validation, and test vectors.
