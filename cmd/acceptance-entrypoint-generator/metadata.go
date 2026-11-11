package main

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

// metadataName maps a feature path to its metadata file name.
func metadataName(featurePath string) string {
	return ""
}

// writeMetadata writes one metadata file per feature into metaDir,
// replacing what was there.
func writeMetadata(features []feature, outDir, irDir, metaDir string) error {
	return nil
}
