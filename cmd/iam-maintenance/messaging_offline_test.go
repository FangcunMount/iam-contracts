package main

import (
	"io"
	"testing"
)

func TestMessagingOfflineRejectsIncompleteApprovalBeforeConnection(t *testing.T) {
	for _, args := range [][]string{
		{}, {"--apply"}, {"--apply", "--writers-excluded"}, {"--apply", "--plan", "missing", "--reviewed-sha256", "wrong"},
		{"--apply", "--writers-excluded", "--plan", "missing", "--reviewed-sha256", "wrong"},
		{"--plan", "missing"}, {"--export-plan", "new", "--writers-excluded"}, {"--timeout", "0s"}, {"--timeout", "10m"}, {"unknown"},
	} {
		if runMessagingOffline(args, io.Discard) == nil {
			t.Fatal("incomplete offline request accepted")
		}
	}
}
