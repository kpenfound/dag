from typing import Annotated, Self
from dagger import Container, dag, Directory, Doc, function, object_type, ReturnType

@object_type
class Workspace:
    """Workspace module for development environments"""
    ctr: Container
    checker: str
    start: Directory
    last_exec_output: str

    @classmethod
    async def create(
        cls,
        base_image: Annotated[str, Doc("Docker base image to use for workspace container")] = "alpine",
        context: Annotated[Directory, Doc("The starting context for the workspace")] = dag.directory(),
        checker: Annotated[str, Doc("The command to check if the workspace meets requirements")] = "echo true"
    ):
        ctr = (
            dag
            .container()
            .from_(base_image)
            .with_workdir("/app")
            .with_directory("/app", context)
        )
        return cls(ctr=ctr, checker=checker, start=context, last_exec_output="")

    @function
    async def read(
        self,
        path: Annotated[str, Doc("File path to read a file from")]
    ) -> str:
        """Returns the contents of a file in the workspace at the provided path"""
        return await self.ctr.file(path).contents()

    @function
    def write_file(
        self,
        path: Annotated[str, Doc("File path to write a file to")],
        contents: Annotated[str, Doc("File contents to write")]
    ) -> Self:
        """Writes the provided contents to a file in the workspace at the provided path"""
        self.ctr = self.ctr.with_new_file(path, contents)
        return self

    @function
    async def write_file_diff(
        self,
        path: Annotated[str, Doc("File path to write a file to")],
        diff: Annotated[str, Doc("Diff content to apply to the file")]
    ) -> Self:
        """Patches a file in the workspace with the provided diff"""
        patch = (
            dag.container().from_("alpine")
            .with_exec(["apk", "add", "patch"])
            .with_file(path, self.ctr.file(path))
            .with_exec(["sh", "-c", f"echo '{diff}' | patch {path}"])
        )
        self.ctr = self.ctr.with_file(path, patch.file(path))
        return self

    @function
    def write_directory(
        self,
        path: Annotated[str, Doc("Directory path to write a directory to")],
        dir: Annotated[Directory, Doc("Directory contents to write")]
    ) -> Self:
        """Writes the provided contents to a directory in the workspace at the provided path"""
        self.ctr = self.ctr.with_directory(path, dir)
        return self

    @function
    async def ls(
        self,
        path: Annotated[str, Doc("Path to get the list of files from")]
    ) -> list[str]:
        """Returns the list of files in the workspace at the provided path"""
        return await self.ctr.directory(path).entries()

    @function
    async def check(
        self
    ) -> str:
        """Checks if the workspace meets the requirements"""
        cmd = (
            self.ctr
            .with_exec(["sh", "-c", self.checker], expect=ReturnType.ANY)
        )
        out = await cmd.stdout() + "\n\n" + await cmd.stderr()
        if await cmd.exit_code() != 0:
            raise Exception(f"Checker failed: {self.checker}\nError: {out}")
        return out

    @function
    async def diff(
        self
    ) -> str:
        """Returns the changes in the workspace so far"""
        start = dag.container().from_("alpine/git").with_workdir("/app").with_directory("/app", self.start)
        # make sure start is a git directory
        if ".git" not in await self.start.entries():
            start = start.with_exec(["git", "init"]).with_exec(["git", "add", "."]).with_exec(["git", "commit", "-m", "'initial'"])
        # return the git diff of the changes in the workspace
        return await start.with_directory(".", self.ctr.directory(".")).with_exec(["git", "diff"]).stdout()

    @function
    async def exec(
        self,
        command: Annotated[str, Doc("command to execute in the workspace")]
    ) -> Self:
        """Executes a command in the workspace. Does not return the output of the command"""
        cmd = (
            self.ctr
            .with_exec(["sh", "-c", command], expect=ReturnType.ANY)
        )
        if await cmd.exit_code() != 0:
            raise Exception(f"Command failed: {command}\nError: {await cmd.stderr()}")
        self.ctr = cmd # FIXME
        self.last_exec_output = await cmd.stdout()
        return self

    @function
    def get_exec_output(
        self
    ) -> str:
        """Returns the output of the last executed command"""
        return self.last_exec_output

    @function
    def container(
        self
    ) -> Container:
        """Returns the container for the workspace"""
        return self.ctr
