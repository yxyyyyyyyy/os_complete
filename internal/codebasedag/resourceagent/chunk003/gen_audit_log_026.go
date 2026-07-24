package chunk003

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"regexp"
	"strings"
	"sync"
	"time"
)

// AuditEntry_026 represents a single auditable command event.
// All fields except Timestamp are validated by ValidateAuditEntry_026.
type AuditEntry_026 struct {
	// Command is the original command string before any redaction.
	Command string `json:"command"`
	// RedactedCommand is the command after secret patterns are replaced.
	RedactedCommand string `json:"redacted_command"`
	// SensitiveFields holds the hashes of any captured secret values.
	SensitiveFields []string `json:"sensitive_fields,omitempty"`
	// Timestamp is the time of the audited event.
	Timestamp time.Time `json:"timestamp"`
	// Severity indicates the importance of the event.
	Severity AuditSeverity_026 `json:"severity"`
	// Result indicates success or failure of the command.
	Result AuditResult_026 `json:"result"`
	// User is the identity that triggered the command.
	User string `json:"user,omitempty"`
	// SessionID correlates related commands.
	SessionID string `json:"session_id,omitempty"`
}

// AuditLog_026 is a thread-safe container for audit entries.
type AuditLog_026 struct {
	mu      sync.RWMutex
	Entries []AuditEntry_026 `json:"entries"`
	MaxSize int              `json:"max_size,omitempty"` // zero means unlimited
}

// AuditSeverity_026 categorises the event importance.
type AuditSeverity_026 int

const (
	AuditSeverity_026_Info    AuditSeverity_026 = iota // 0
	AuditSeverity_026_Warning                          // 1
	AuditSeverity_026_Error                            // 2
	AuditSeverity_026_Critical                         // 3
)

// AuditResult_026 indicates the outcome of an audited action.
type AuditResult_026 int

const (
	AuditResult_026_Unknown  AuditResult_026 = iota // 0
	AuditResult_026_Success                         // 1
	AuditResult_026_Failure                         // 2
	AuditResult_026_Partial                         // 3
)

// SecretType_026 enumerates known secret categories.
type SecretType_026 int

const (
	SecretType_026_Unknown   SecretType_026 = iota
	SecretType_026_Password                  // 1
	SecretType_026_APIToken                  // 2
	SecretType_026_SSHKey                    // 3
	SecretType_026_AWSKey                    // 4
	SecretType_026_GenericSecret             // 5
)

// String returns the human-readable name of a SecretType_026.
func (s SecretType_026) String() string {
	switch s {
	case SecretType_026_Password:
		return "password"
	case SecretType_026_APIToken:
		return "api_token"
	case SecretType_026_SSHKey:
		return "ssh_key"
	case SecretType_026_AWSKey:
		return "aws_key"
	case SecretType_026_GenericSecret:
		return "generic_secret"
	default:
		return "unknown"
	}
}

// SecretPattern_026 holds a compiled regular expression and a placeholder
// used to redact secrets in command strings.
type SecretPattern_026 struct {
	Regexp      *regexp.Regexp
	Placeholder string
	SecretType  SecretType_026
}

// redactionPatterns_026 is a deterministic table of known secret patterns.
// Each entry provides a regexp to capture a secret value and a placeholder
// to replace it with.
var redactionPatterns_026 = []SecretPattern_026{
	{
		Regexp:      regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
		Placeholder: "password=***",
		SecretType:  SecretType_026_Password,
	},
	{
		Regexp:      regexp.MustCompile(`(?i)(token|api_key|apikey)\s*[:=]\s*\S+`),
		Placeholder: "token=***",
		SecretType:  SecretType_026_APIToken,
	},
	{
		Regexp:      regexp.MustCompile(`(?i)(-----BEGIN[ A-Z]*PRIVATE KEY-----)?.+?(-----END[ A-Z]*PRIVATE KEY-----)`),
		Placeholder: "ssh_private_key=***",
		SecretType:  SecretType_026_SSHKey,
	},
	{
		Regexp:      regexp.MustCompile(`(?i)(aws_access_key_id|aws_secret_access_key)\s*[:=]\s*\S+`),
		Placeholder: "aws_key=***",
		SecretType:  SecretType_026_AWSKey,
	},
	{
		Regexp:      regexp.MustCompile(`(?i)(secret|secret_key)\s*[:=]\s*\S+`),
		Placeholder: "secret=***",
		SecretType:  SecretType_026_GenericSecret,
	},
}

// hashAlgorithms_026 maps algorithm names to their hash.Hash constructors.
// This table is used to deterministically hash sensitive fields.
var hashAlgorithms_026 = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha512": sha512.New,
}

// DefaultHashAlg_026 is the fallback algorithm if none is specified.
const DefaultHashAlg_026 = "sha256"

// ResourceAgentDefaultHashAlg_026 is the exported constant for default hash.
const ResourceAgentDefaultHashAlg_026 = DefaultHashAlg_026

// NewAuditEntry_026 creates a new AuditEntry_026, redacts secrets and
// hashes captured sensitive fields.
func NewAuditEntry_026(cmd string, severity AuditSeverity_026, result AuditResult_026, user, sessionID string) (AuditEntry_026, error) {
	if strings.TrimSpace(cmd) == "" {
		return AuditEntry_026{}, errors.New("command must not be empty")
	}
	entry := AuditEntry_026{
		Command:   cmd,
		Timestamp: time.Now().UTC(),
		Severity:  severity,
		Result:    result,
		User:      user,
		SessionID: sessionID,
	}
	entry.RedactedCommand, entry.SensitiveFields = RedactSecrets_026(cmd, DefaultHashAlg_026)
	return entry, nil
}

// RedactSecrets_026 scans the command string, replaces known secret patterns
// with placeholders, and returns the redacted string along with hashes of
// any discovered secret values. The algorithm parameter determines which
// hash function is used (must exist in hashAlgorithms_026).
func RedactSecrets_026(cmd, hashAlg string) (redacted string, hashes []string) {
	redacted = cmd
	for _, pat := range redactionPatterns_026 {
		// Replace all occurrences with the placeholder.
		// We also capture the original matched secret value for hashing.
		redacted = pat.Regexp.ReplaceAllStringFunc(redacted, func(match string) string {
			// Hash the entire matched substring (including key=value)
			h, err := HashSensitiveField_026(match, hashAlg)
			if err == nil {
				hashes = append(hashes, h)
			}
			return pat.Placeholder
		})
	}
	return
}

// HashSensitiveField_026 hashes a value using the given algorithm name.
// If the algorithm is unknown, it falls back to DefaultHashAlg_026 and
// returns the hash with a non-nil error.
func HashSensitiveField_026(value, alg string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("cannot hash empty value")
	}
	newHash, ok := hashAlgorithms_026[alg]
	if !ok {
		newHash = hashAlgorithms_026[DefaultHashAlg_026]
		h := newHash()
		h.Write([]byte(value))
		return hex.EncodeToString(h.Sum(nil)), fmt.Errorf("unknown hash algorithm %q, using %s", alg, DefaultHashAlg_026)
	}
	h := newHash()
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ValidateAuditEntry_026 checks an AuditEntry_026 for required fields and
// valid enum values. It returns an error describing the first problem.
func ValidateAuditEntry_026(entry AuditEntry_026) error {
	if strings.TrimSpace(entry.Command) == "" {
		return errors.New("command must not be empty")
	}
	if strings.TrimSpace(entry.RedactedCommand) == "" {
		return errors.New("redacted_command must not be empty")
	}
	if entry.Timestamp.IsZero() {
		return errors.New("timestamp must be set")
	}
	if entry.Severity < AuditSeverity_026_Info || entry.Severity > AuditSeverity_026_Critical {
		return fmt.Errorf("invalid severity: %d", entry.Severity)
	}
	if entry.Result < AuditResult_026_Unknown || entry.Result > AuditResult_026_Partial {
		return fmt.Errorf("invalid result: %d", entry.Result)
	}
	// SensitiveFields may be empty; that's acceptable.
	return nil
}

// ValidateAuditLog_026 checks an AuditLog_026 for consistency.
func ValidateAuditLog_026(log *AuditLog_026) error {
	if log == nil {
		return errors.New("audit log is nil")
	}
	for i, entry := range log.Entries {
		if err := ValidateAuditEntry_026(entry); err != nil {
			return fmt.Errorf("entry %d: %w", i, err)
		}
	}
	return nil
}

// AppendEntry_026 adds an entry to the AuditLog_026, respecting MaxSize.
func (l *AuditLog_026) AppendEntry_026(entry AuditEntry_026) error {
	if err := ValidateAuditEntry_026(entry); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.MaxSize > 0 && len(l.Entries) >= l.MaxSize {
		// Evict oldest entry (FIFO)
		l.Entries = l.Entries[1:]
	}
	l.Entries = append(l.Entries, entry)
	return nil
}

// MarshalJSON_026 serializes the AuditLog_026 to JSON in a thread-safe way.
func (l *AuditLog_026) MarshalJSON_026() ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return json.Marshal(l)
}

// UnmarshalJSON_026 deserializes JSON into the AuditLog_026.
func (l *AuditLog_026) UnmarshalJSON_026(data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return json.Unmarshal(data, l)
}

// ResourceAgentNewAuditLog_026 creates a new AuditLog_026 with optional max size.
func ResourceAgentNewAuditLog_026(maxSize int) *AuditLog_026 {
	return &AuditLog_026{
		Entries: make([]AuditEntry_026, 0),
		MaxSize: maxSize,
	}
}

// ResourceAgentValidAuditEntry_026 returns a deterministic valid AuditEntry_026
// for testing purposes.
func ResourceAgentValidAuditEntry_026() AuditEntry_026 {
	entry, _ := NewAuditEntry_026(
		"deploy --password s3cr3t --token abc123",
		AuditSeverity_026_Info,
		AuditResult_026_Success,
		"ops",
		"session-001",
	)
	return entry
}

// ResourceAgentRedactionPatterns_026 returns a copy of the redaction patterns table.
func ResourceAgentRedactionPatterns_026() []SecretPattern_026 {
	patterns := make([]SecretPattern_026, len(redactionPatterns_026))
	copy(patterns, redactionPatterns_026)
	return patterns
}

// ResourceAgentHashAlgorithms_026 returns a copy of the supported hash algorithms.
func ResourceAgentHashAlgorithms_026() map[string]func() hash.Hash {
	algs := make(map[string]func() hash.Hash, len(hashAlgorithms_026))
	for k, v := range hashAlgorithms_026 {
		algs[k] = v
	}
	return algs
}

// SeverityString returns the string representation of an AuditSeverity_026.
func (s AuditSeverity_026) SeverityString() string {
	switch s {
	case AuditSeverity_026_Info:
		return "INFO"
	case AuditSeverity_026_Warning:
		return "WARNING"
	case AuditSeverity_026_Error:
		return "ERROR"
	case AuditSeverity_026_Critical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ResultString returns the string representation of an AuditResult_026.
func (r AuditResult_026) ResultString() string {
	switch r {
	case AuditResult_026_Success:
		return "SUCCESS"
	case AuditResult_026_Failure:
		return "FAILURE"
	case AuditResult_026_Partial:
		return "PARTIAL"
	default:
		return "UNKNOWN"
	}
}

// RedactSecretsWithCustomPatterns_026 allows using an external pattern set.
func RedactSecretsWithCustomPatterns_026(cmd, hashAlg string, patterns []SecretPattern_026) (string, []string) {
	redacted := cmd
	var hashes []string
	for _, pat := range patterns {
		redacted = pat.Regexp.ReplaceAllStringFunc(redacted, func(match string) string {
			h, err := HashSensitiveField_026(match, hashAlg)
			if err == nil {
				hashes = append(hashes, h)
			}
			return pat.Placeholder
		})
	}
	return redacted, hashes
}

// ResourceAgentRedactSecrets_026 is an exported alias for RedactSecrets_026.
func ResourceAgentRedactSecrets_026(cmd, hashAlg string) (string, []string) {
	return RedactSecrets_026(cmd, hashAlg)
}

// ResourceAgentValidateAuditEntry_026 is an exported Validate function.
func ResourceAgentValidateAuditEntry_026(entry AuditEntry_026) error {
	return ValidateAuditEntry_026(entry)
}

// ResourceAgentValidateAuditLog_026 is an exported Validate function for logs.
func ResourceAgentValidateAuditLog_026(log *AuditLog_026) error {
	return ValidateAuditLog_026(log)
}

// init registers the default hash algorithm in the table for safety.
func init() {
	// Ensure at least sha256 always present.
	if _, ok := hashAlgorithms_026["sha256"]; !ok {
		hashAlgorithms_026["sha256"] = sha256.New
	}
}
