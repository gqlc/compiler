package compiler

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	// Create dummy GenCtx for this test
	type testCtx struct {
		GeneratorContext
	}

	// Prepare a context
	ctx := WithContext(context.Background(), &testCtx{})

	// Check that a normal string can't retrieve value
	if ctx.Value("genCtx") != nil {
		t.Fail()
		return
	}

	// Check that Context properly retrieves
	if Context(ctx) == nil {
		t.Fail()
	}
}
