/**
 * A generated module for playwright functions
 */
import {
  dag,
  argument,
  Directory,
  object,
  func,
  check,
} from "@dagger.io/dagger";

@object()
export class Playwright {
  /**
   * Runs the playwright tests
   */
  @func()
  @check()
  test(
    @argument({ defaultPath: "/" }) source: Directory,
    task: string = "test:e2e",
  ): Promise<string> {
    return dag
      .container()
      .from("node:18")
      .withWorkdir("/src")
      .withDirectory("/src", source)
      .withEnvVariable("CI", "true")
      .withExec(["npm", "ci"])
      .withExec(["npx", "playwright", "install", "--with-deps"])
      .withExec(["npm", "run", task])
      .stdout();
    // .file("playwright-report")
    // .contents();
  }
}
