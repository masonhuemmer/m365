package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMetadataNameFollowsAPS(t *testing.T) {
	cases := map[string]string{
		"features/Hunt The Wumpus.feature":     "features-hunt-the-wumpus-feature.json",
		"features/orders/Cancel Order.feature": "features-orders-cancel-order-feature.json",
		"Features/API v2/Happy Path.feature":   "features-api-v2-happy-path-feature.json",
		"features/cli/exit-codes.feature":      "features-cli-exit-codes-feature.json",
	}
	for path, want := range cases {
		if got := metadataName(path); got != want {
			t.Errorf("metadataName(%q) = %q, want %q", path, got, want)
		}
	}
}

func readMetadata(t *testing.T, path string) metadata {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m metadata
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func sha256Of(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestWriteMetadataDescribesEachGeneratedTest(t *testing.T) {
	t.Chdir(layout(t, map[string]string{"features/cli/help.feature": "Feature: cli help\n"}))
	if err := generate("features", "acceptance/generated"); err != nil {
		t.Fatal(err)
	}
	features, err := listFeatures("features")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(features, "acceptance/generated", "build/acceptance/ir", "build/acceptance/metadata"); err != nil {
		t.Fatal(err)
	}
	got := readMetadata(t, "build/acceptance/metadata/features-cli-help-feature.json")
	want := metadata{
		SchemaVersion:      1,
		FeaturePath:        "features/cli/help.feature",
		IRPath:             "build/acceptance/ir/cli_help.json",
		ImplementationHash: sha256Of(t, "acceptance/generated/cli_help_acceptance_test.go"),
		HashScope:          "generated_files",
		GeneratedFiles:     []string{"acceptance/generated/cli_help_acceptance_test.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata\n%+v\nwant\n%+v", got, want)
	}
}

func TestWriteMetadataDropsStaleFiles(t *testing.T) {
	t.Chdir(layout(t, map[string]string{
		"features/cli/help.feature":                   "Feature: cli help\n",
		"build/acceptance/metadata/features-old.json": "{}\n",
	}))
	if err := generate("features", "acceptance/generated"); err != nil {
		t.Fatal(err)
	}
	features, err := listFeatures("features")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(features, "acceptance/generated", "build/acceptance/ir", "build/acceptance/metadata"); err != nil {
		t.Fatal(err)
	}
	if got := generated(t, "build/acceptance/metadata"); !reflect.DeepEqual(got, []string{"features-cli-help-feature.json"}) {
		t.Fatalf("metadata files %v, want only the current feature's", got)
	}
}

func TestRunWritesMetadataBesideTheIR(t *testing.T) {
	t.Chdir(layout(t, map[string]string{"features/mail/send-reply.feature": "Feature: send\n"}))
	if err := run(false); err != nil {
		t.Fatal(err)
	}
	m := readMetadata(t, filepath.Join("build", "acceptance", "metadata", "features-mail-send-reply-feature.json"))
	if m.FeaturePath != "features/mail/send-reply.feature" || m.IRPath != "build/acceptance/ir/mail_send_reply.json" {
		t.Fatalf("run wrote %+v", m)
	}
}
