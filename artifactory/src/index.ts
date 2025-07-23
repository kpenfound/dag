/**
 * A module for Artifactory functions
 *
 * This module provides functions for uploading build artifacts and related files to your Artifactory instance
 */

import {
  dag,
  Container,
  Directory,
  File,
  Secret,
  object,
  func,
} from "@dagger.io/dagger";

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
    cliVersion: string = "latest",
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
    file: File,
    targetPath: string,
    flags: string[],
  ): Promise<string> {
    let command = await this.jfExec(
      ["rt", "u", "/artifact", targetPath, ...flags],
      file,
    );

    return command.stdout();
  }

  /**
   * Upload evidence to an artifact
   */
  @func()
  async uploadArtifactEvidence(): Promise<string> {
    return dag.container().stdout();
  }

  /**
   * Upload evidence to a package
   */
  @func()
  async uploadPackageEvidence(): Promise<string> {
    return dag.container().stdout();
  }

  /**
   * Upload evidence to a build
   */
  @func()
  async uploadBuildEvidence(): Promise<string> {
    return dag.container().stdout();
  }

  /**
   * Upload evidence to a release bundle
   */
  @func()
  async uploadReleaseBundleEvidence(): Promise<string> {
    return dag.container().stdout();
  }

  /**
   * Run a command with the jf CLI
   */
  @func()
  async jfExec(args: string[], file?: File): Promise<Container> {
    let cli = this.cli();
    if (file) {
      cli = cli.withFile("/artifact", file);
    }
    return cli.withExec([
      "jf",
      "--url",
      this.url,
      "--access-token",
      await this.accessToken.plaintext(),
      ...args,
    ]);
  }

  /**
   * Get a Container with the jf CLI installed
   */
  @func()
  cli(): Container {
    const base = dag.container().from("alpine:3");
    return dag.jfrogcli({ version: this.cliVersion }).install({ base: base });
  }
}
