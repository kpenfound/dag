import { dag, object, func, Secret } from "@dagger.io/dagger";

@object()
export class Example {
  /**
   * Upload an artifact with evidence
   */
  @func()
  async example_uploadArtifactWithEvidence(
    instanceUrl: string,
    accessToken: Secret,
  ): Promise<string> {
    // Construct the artifactory module
    const client = dag.artifactory(accessToken, instanceUrl);

    // Upload an artifact
    const artifact = dag.http(
      "https://github.com/dagger/dagger/releases/download/v0.18.13/dagger_v0.18.13_linux_amd64.tar.gz",
    );
    const artifactPath = "generic-repo/dagger_v0.18.13_linux_amd64.tar.gz";
    const uploadOutput = await client.upload(artifact, artifactPath);

    // Wait a moment for upload to process
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // Upload evidence for the artifact
    const predicate = dag
      .currentModule()
      .source()
      .file("./test_predicate.json");
    const predicateType = "https://in-toto.io/Statement/v1";
    const key = dag.currentModule().source().file("./private.pem");
    const keyAlias = "OtherKey";
    const traceUrl = await dag.cloud().traceURL();
    const createEvidenceOutput = await client.createEvidence(
      predicate,
      predicateType,
      key,
      { keyAlias: keyAlias, subjectRepoPath: artifactPath, traceUrl: traceUrl },
    );

    return uploadOutput + "\n" + createEvidenceOutput;
  }
}
