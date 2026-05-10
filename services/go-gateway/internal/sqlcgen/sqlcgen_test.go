package sqlcgen

import (
	"testing"
)

func TestQueriesHealthCheckExists(t *testing.T) {
	var q *Queries
	_ = q.HealthCheck
	_ = q.WithTx
	_ = New
}