package main

import (
	"context"
	"fmt"
	"time"

	"dagger/example/internal/dagger"
)

type Example struct{}

// Upload an artifact with evidence
func (m *Example) Example_UploadArtifactWithEvidence(
	ctx context.Context,
	instanceURL string,
	accessToken *dagger.Secret,
) (string, error) {
	// Construct the artifactory module
	client := dag.Artifactory(accessToken, instanceURL)

	// Upload an artifact
	artifact := dag.HTTP(
		"https://github.com/dagger/dagger/releases/download/v0.18.13/dagger_v0.18.13_linux_amd64.tar.gz",
	)
	artifactPath := "generic-repo/dagger_v0.18.13_linux_amd64.tar.gz"
	uploadOutput, err := client.Upload(ctx, artifact, artifactPath)
	if err != nil {
		return "", err
	}

	// Wait a moment for upload to process
	time.Sleep(1 * time.Second)

	// Upload evidence for the artifact
	predicate := dag.
		CurrentModule().
		Source().
		File("./test_predicate.json")
	predicateType := "https://in-toto.io/Statement/v1"
	key := dag.CurrentModule().Source().File("./private.pem")
	keyAlias := "OtherKey"
	traceURL, err := dag.Cloud().TraceURL(ctx)
	if err != nil {
		return "", err
	}

	createEvidenceOutput, err := client.CreateEvidence(
		ctx,
		predicate,
		predicateType,
		key,
		dagger.ArtifactoryCreateEvidenceOpts{
			KeyAlias:        keyAlias,
			SubjectRepoPath: artifactPath,
			TraceURL:        traceURL,
		})

	return fmt.Sprintf("%s\n%s", uploadOutput, createEvidenceOutput), nil
}
