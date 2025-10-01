package tests

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestEcosystemAuditBehaviorSuite runs the ecosystem audit behavior test suite
func TestEcosystemAuditBehaviorSuite(t *testing.T) {
	suite.Run(t, new(BehaviorTestSuite))
}