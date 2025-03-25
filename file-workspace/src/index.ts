/**
 * A workspace for reading and writing files in a project
 */
import { dag, Container, Directory, object, func } from "@dagger.io/dagger";

@object()
export class FileWorkspace {
  /**
   * The source directory of the project
   */
  @func()
  source: Directory;

  constructor(source: Directory) {
    this.source = source;
  }

  /**
   * Read a file from the project
   * @param path The path to the file
   */
  @func()
  async read(path: string): Promise<string> {
    return this.source.file(path).contents();
  }

  /**
   * Write a contents to a file in the project
   * @param path The path to the file
   * @param content The content to write in the file
   */
  @func()
  write(path: string, content: string): FileWorkspace {
    this.source = this.source.withNewFile(path, content);
    return this;
  }

  /**
   * List the files in the workspace in tree format
   */
  @func()
  async listFiles(): Promise<string> {
    return dag
      .container()
      .from("alpine")
      .withDirectory("/app", this.source)
      .withExec(["tree", "/app"])
      .stdout();
  }
}
