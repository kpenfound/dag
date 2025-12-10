/**
 * A module for Artifactory functions
 *
 * This module provides functions for uploading build artifacts and related files to your Artifactory instance
 */

import { dag, Container, File, Secret, object, func } from "@dagger.io/dagger";

@object()
export class Artifactory {
  accessToken: Secret;
  cliVersion: string;
  url: string;

  constructor(
    /**
     * Artifactory access token
     */
    accessToken: Secret,
    /**
     * Jfrog Artifactory API URL. It usually ends with /artifactory
     */
    url: string,
    /**
     * Jfrog CLI version
     */
    cliVersion: string = "2.78.0",
  ) {
    this.accessToken = accessToken;
    this.cliVersion = cliVersion;
    this.url = url;
  }

  /**
   * Upload any file to artifactory
   */
  @func()
  async upload(
    /**
     * file to upload
     */
    file: File,
    /**
     * path in artifactory to upload to
     */
    targetPath: string,
    /**
     * extra arguments for jf cli
     */
    extraFlags?: string[],
    /**
     * key to sign dagger evidence
     */
    evidenceKey?: File,
    /**
     * key alias for evidence signing key
     */
    evidenceKeyAlias?: string,
  ): Promise<string> {
    // assemble execution container
    let cli = await this.cli();

    cli = cli.withFile("/artifact", file);
    let args = ["jf", "rt", "u", "/artifact", targetPath];
    if (extraFlags) {
      args.push(...extraFlags);
    }

    if (evidenceKey) {
      // automatically attach evidence from dagger cloud trace
      const uploadOutput = await cli.withExec(args).stdout();
      const traceUrl = await dag.cloud().traceURL();

      // figure out what kind of artifact this is
      let predicate = dag.file("foo", "bar");
      let predicateType = "bar";
      const evidenceOut = await this.createEvidence(
        predicate,
        predicateType,
        evidenceKey,
        "dagger", // provider-id
        evidenceKeyAlias,
        targetPath,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        traceUrl,
      );
      return uploadOutput + evidenceOut;
    }

    return cli.withExec(args).stdout();
  }

  /**
   * Upload evidence to a resource in Artifactory
   */
  @func()
  async createEvidence(
    /**
     * predicate File to attach as evidence
     */
    predicate: File,
    /**
     * predicate type based on in-toto attestation predicates
     */
    predicateType: string,
    /**
     * private key to sign DSSE envelope
     */
    key: File,
    /**
     * name of the provider that created the evidence
     */
    providerId: string = "dagger",
    /**
     * key alias for a public key that exists in your Artifactory instance to validate the signature
     */
    keyAlias?: string,
    /**
     * If the resource is an Artifact, a path to the artifact
     */
    subjectRepoPath?: string,
    /**
     * If the resource is a Package, the name of the package
     */
    packageName?: string,
    /**
     * If the resource is a Package, the version of the package
     */
    packageVersion?: string,
    /**
     * If the resource is a Package, the name of the package repo
     */
    packageRepoName?: string,
    /**
     * If the resource is a Build, the name of the build
     */
    buildName?: string,
    /**
     * If the resource is a Build, the number of the build
     */
    buildNumber?: string,
    /**
     * If the resource is a Release Bundle, the name of the release bundle
     */
    releaseBundleName?: string,
    /**
     * If the resource is a Release Bundle, the version of the release bundle
     */
    releaseBundleVersion?: string,
    /**
     * extra flags to pass through to jf cli
     */
    extraArgs?: string[],
    /**
     * dagger cloud trace URL to attach to evidence
     */
    traceUrl?: string,
  ): Promise<string> {
    // assemble execution container
    let cli = await this.cli();

    // assemble args
    let args = ["jf", "evd", "create"];
    // --predicate
    cli = cli.withFile("/predicate.json", predicate);
    args.push("--predicate", "/predicate.json");
    // --predicate-type
    args.push("--predicate-type", predicateType);
    // --key
    cli = cli.withFile("/key.pem", key);
    args.push("--key", "/key.pem");
    // --key-alias
    if (keyAlias) {
      args.push("--key-alias", keyAlias);
    }
    if (providerId) {
      args.push("--provider-id", providerId);
    }
    // --markdown
    if (traceUrl) {
      const markdown = `## View the trace on Dagger Cloud\n\n[Dagger Cloud Trace](${traceUrl})`;
      cli = cli.withNewFile("/daggerCloud.md", markdown);
      args.push("--markdown", "/daggerCloud.md");
    }

    // evidence subject

    // --subject-repo-path
    if (subjectRepoPath) {
      args.push("--subject-repo-path", subjectRepoPath);
    }

    // Package information
    if (packageName) {
      args.push("--package-name", packageName);
    }
    if (packageVersion) {
      args.push("--package-version", packageVersion);
    }
    if (packageRepoName) {
      args.push("--package-repo-name", packageRepoName);
    }

    // Build information
    if (buildName) {
      args.push("--build-name", buildName);
    }
    if (buildNumber) {
      args.push("--build-number", buildNumber);
    }

    // Release Bundle information
    if (releaseBundleName) {
      args.push("--release-bundle-name", releaseBundleName);
    }
    if (releaseBundleVersion) {
      args.push("--release-bundle-version", releaseBundleVersion);
    }

    // user defined args
    if (extraArgs) {
      args.push(...extraArgs);
    }

    // execute the command
    return cli.withExec(args).stdout();
  }

  /**
   * Run a command with the jf CLI
   */
  @func()
  async jfExec(
    /**
     * arguments to pass to jf cli
     */
    args: string[],
  ): Promise<Container> {
    let cli = await this.cli();
    return cli.withExec(args);
  }

  /**
   * Get a Container with the jf CLI installed
   */
  @func()
  async cli(): Promise<Container> {
    const base = dag.container().from("alpine:3");
    return dag
      .jfrogcli({ version: this.cliVersion })
      .install({ base: base })
      .withSecretVariable("JFROG_ACCESS_TOKEN", this.accessToken)
      .withExec([
        "sh",
        "-c",
        `jf config add --url ${this.url} --access-token $JFROG_ACCESS_TOKEN`,
      ]);
  }
}
