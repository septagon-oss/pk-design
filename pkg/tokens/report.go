package tokens

// report.go owns validation report behavior for token consumers and tooling.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strings"
)

// Add appends an issue.
func (r *Report) Add(code IssueCode, severity Severity, path, message string) {
	r.Issues = append(r.Issues, Issue{
		Code:     code,
		Severity: severity,
		Path:     strings.TrimSpace(path),
		Message:  strings.TrimSpace(message),
	})
}

// HasErrors reports whether any validation issue has error severity.
func (r Report) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Error returns a compact summary of the report's error issues.
func (r Report) Error() string {
	if len(r.Issues) == 0 {
		return "token validation failed"
	}
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			if issue.Path != "" {
				return fmt.Sprintf("%s at %s: %s", issue.Code, issue.Path, issue.Message)
			}
			return fmt.Sprintf("%s: %s", issue.Code, issue.Message)
		}
	}
	return "token validation completed with warnings"
}
