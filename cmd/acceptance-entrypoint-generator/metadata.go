package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// metadata is the APS generator metadata for one feature
// (acceptance-generator.md, "Generator Metadata").
type metadata struct {
	SchemaVersion      int      `json:"schema_version"`
	FeaturePath        string   `json:"feature_path"`
	IRPath             string   `json:"ir_path"`
	ImplementationHash string   `json:"implementation_hash"`
	HashScope          string   `json:"hash_scope"`
	GeneratedFiles     []string `json:"generated_files"`
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// metadataName maps a feature path to its metadata file name: lowercase,
// each run of non-alphanumerics as one hyphen, trimmed, plus ".json".
func metadataName(featurePath string) string {
	s := nonAlnum.ReplaceAllString(strings.ToLower(filepath.ToSlash(featurePath)), "-")
	return strings.Trim(s, "-") + ".json"
}

// writeMetadata writes one metadata file per feature into metaDir,
// replacing what was there. The implementation hash covers the feature's
// generated test file only.
func writeMetadata(features []feature, outDir, irDir, metaDir string) error {
	if err := os.RemoveAll(metaDir); err != nil {
		return err
	}
	if err := os.MkdirAll(metaDir, 0o750); err != nil {
		return err
	}
	for _, f := range features {
		if err := writeFeatureMetadata(f, outDir, irDir, metaDir); err != nil {
			return err
		}
	}
	return nil
}

func writeFeatureMetadata(f feature, outDir, irDir, metaDir string) error {
	gen := filepath.ToSlash(filepath.Join(outDir, f.Name+generatedSuffix))
	src, err := os.ReadFile(gen) // #nosec G304 -- the test generate just wrote
	if err != nil {
		return err
	}
	sum := sha256.Sum256(src)
	m := metadata{
		SchemaVersion:      1,
		FeaturePath:        filepath.ToSlash(f.Path),
		IRPath:             filepath.ToSlash(filepath.Join(irDir, f.Name+".json")),
		ImplementationHash: "sha256:" + hex.EncodeToString(sum[:]),
		HashScope:          "generated_files",
		GeneratedFiles:     []string{gen},
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(metaDir, metadataName(m.FeaturePath)), append(out, '\n'), 0o600)
}
