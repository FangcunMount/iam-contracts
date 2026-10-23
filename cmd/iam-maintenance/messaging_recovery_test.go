package main

import (
	"io"
	"testing"
)

func TestMessagingRecoveryRequiresOriginalScopeAndExternalAttestations(t *testing.T) {
	for _, args := range [][]string{{}, {"--restore"}, {"--restore", "--writers-excluded"},
		{"--restore", "--writers-excluded", "--pre-start-restore-attested", "--protection", "missing", "--reviewed-sha256", "wrong"},
		{"--export-protection", "new", "--writers-excluded"}, {"--timeout", "0s"}, {"--timeout", "6m"},
	} {
		if runMessagingRecovery(args, io.Discard) == nil {
			t.Fatal("incomplete protection accepted")
		}
	}
}
