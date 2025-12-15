package vault

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var auditlog = logf.Log.WithName("audit")

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Who     string                 `json:"who"`
	When    time.Time              `json:"when"`
	What    string                 `json:"what"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// parseJWT extracts the subject (who) from a JWT token without verification.
// Parameters:
// - tokenString: the JWT token as a string
// Returns: the subject claim or "unknown" if not found, and an error if parsing fails
func parseJWT(tokenString string) (string, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
	}

	return "unknown", nil
}

// LogAudit logs an audit entry in structured format.
// Parameters:
// - jwtToken: the JWT token used to extract the user identity
// - action: the action being audited
// - context: additional context information for the audit log
func LogAudit(jwtToken string, action string, context map[string]interface{}) {
	who, err := parseJWT(jwtToken)
	if err != nil {
		auditlog.Error(err, "Failed to parse JWT for audit")
		who = "unknown"
	}

	when := time.Now()

	auditlog.Info(action, "user", who, "timestamp", when.Format(time.RFC3339), "context", context)
}
