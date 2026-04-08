package main

import "testing"

func TestApplyMockModeAppendsFamilyInit(t *testing.T) {
	steps := applyMockMode([]seedStep{
		stepSystemInit,
		stepAuthnInit,
		stepTenantBootstrapAdmin,
		stepAdminInit,
	}, true)

	if len(steps) != 5 {
		t.Fatalf("applyMockMode() len = %d, want 5", len(steps))
	}
	if steps[len(steps)-1] != stepFamilyInit {
		t.Fatalf("applyMockMode() last step = %q, want %q", steps[len(steps)-1], stepFamilyInit)
	}
}

func TestApplyMockModeDoesNotDuplicateFamilyInit(t *testing.T) {
	steps := applyMockMode([]seedStep{
		stepSystemInit,
		stepFamilyInit,
	}, true)

	count := 0
	for _, step := range steps {
		if step == stepFamilyInit {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("applyMockMode() family-init count = %d, want 1", count)
	}
}
