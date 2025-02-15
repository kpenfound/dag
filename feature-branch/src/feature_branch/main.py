from typing import Annotated, Self
from dagger import Container, dag, Directory, Doc, field, function, object_type, Secret

gh_cli_version = "2.66.1"
git_container = "alpine/git:v2.47.1"

@object_type
class FeatureBranch:
    """Feature Branch module for GitHub development"""
    github_token: Annotated[Secret, Doc("GitHub Token")] | None = field(default=None)
    branch_name: str | None
    is_fork: bool = False
    branch: Annotated[Directory, Doc("A git repo")] | None = field(default=dag.directory())
    @classmethod
    async def create(
        self,
        upstream: str,
        branch_name: str,
        fork_name: str | None,
        fork: bool = False
    ) -> Self:
        self.is_fork = fork
        self.branch_name = branch_name
        self.branch = dag.git(upstream).head().tree()
        if fork:
            self = await self.fork(upstream, fork_name)
        self.branch = (
            self.env()
            .with_exec([
                "git",
                "checkout",
                "-b",
                branch_name,
            ])
            .directory(".")
        )
        return self

    @function
    def with_changes(
        self,
        changes: Directory
    ) -> Self:
        """Apply a directory of changes to the branch"""
        self.branch = self.branch.with_directory(".", changes)
        return self

    @function
    def with_github_token(
        self,
        token: Secret
    ) -> Self:
        """Sets the GitHub token"""
        self.github_token = token
        return self

    @function
    async def pull_request(
        self,
        title: str,
        body: str
    ) -> str:
        """Creates a pull request on the branch with the provided title and body"""
        origin = await self.get_remote_url("origin")
        upstream = origin
        if self.is_fork:
            upstream = await self.get_remote_url("upstream")

        return await (
            self.env()
            .with_exec([
                "git",
                "add",
                ".",
            ])
            .with_exec([
                "git",
                "commit",
                "-m",
                f"'{title}'",
            ])
            .with_exec([
                "git",
                "push",
                "origin",
                self.branch_name,
            ])
            .with_exec([
                "gh",
                "pr",
                "create",
                "--title",
                f"'{title}'",
                "--body",
                f"'{body}'",
                "--repo",
                upstream,
                "--base",
                "main",
                "--head",
                f"{origin.split('/')[-2]}:{self.branch_name}",
            ])
            .stdout()
        )

    @function
    async def get_remote_url(
        self,
        remote: str
    ) -> str:
        """Returns a remotes url"""
        remote = await (
            self.env()
            .with_exec(["git", "remote", "get-url", remote])
            .stdout()
        )
        remote = str.removesuffix(remote, "\n")
        remote = str.removesuffix(remote, ".git")
        remote = str.removeprefix(remote, "https://")
        return remote

    @function
    def env(self) -> Container:
        return (
            dag.github(gh_cli_version)
            .container(
                dag.container().from_(git_container)
            )
            .with_secret_variable("GITHUB_TOKEN", self.github_token)
            .with_workdir("/src")
            .with_mounted_directory("/src", self.branch)
            .with_exec(["gh", "auth", "setup-git"])
            .with_exec(["git", "config", "--global", "user.name", "Marvin"])
            .with_exec(["git", "config", "--global", "user.email", "marvin@dagger.io"])
        )

    @function
    async def fork(
        self,
        upstream: str,
        fork_name: str | None
    ) -> Self:
        """Forks a repository"""
        if fork_name is None:
            fork_name = upstream.split("/")[-1] + "-fork"

        # check for fork
        result = await (
            self.env()
            .with_exec([
                "sh",
                "-c",
                f"gh repo list --json name --jq '.[] | .name' | grep '{fork_name}'",
            ])
            .stdout()
        )

        # We have already forked this repository. Just clone it
        if result == fork_name:
            self.branch = (
                self.env()
                .with_exec([
                    "gh",
                    "repo",
                    "clone",
                    fork_name,
                ]).directory(fork_name)
            )
            return self

        # Fork it
        self.branch = (
            self.env()
            .with_exec([
                "gh",
                "repo",
                "fork",
                upstream,
                "--default-branch-only",
                "--clone",
                "--fork-name",
                fork_name,
            ]).directory(fork_name)
        )
        return self
