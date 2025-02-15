from typing import Annotated, Self
from dagger import Container, dag, Directory, Doc, function, object_type, ReturnType

@object_type
class Workspace:
    """Workspace module for development environments"""
    ctr: Container
    checker: str

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
        return cls(ctr=ctr, checker=checker)

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
    ) -> bool:
        """Checks if the workspace meets the requirements"""
        cmd = (
            self.ctr
            .with_exec(["sh", "-c", self.checker], expect=ReturnType.ANY)
        )
        return await cmd.exit_code() == 0


    @function
    async def exec(
        self,
        command: Annotated[str, Doc("command to execute in the workspace")]
    ) -> Self:
        """Executes a command in the workspace"""
        cmd = (
            self.ctr
            .with_exec(["sh", "-c", command], expect=ReturnType.ANY)
        )
        if await cmd.exit_code() != 0:
            raise Exception(f"Command failed: {command}\nError: {await cmd.stderr()}")
        self.ctr = cmd # FIXME
        return self
