from typing import Annotated, Self
from dagger import Container, dag, Directory, Doc, field, function, object_type, Secret
import time

gh_cli_version = "2.66.1"
git_container = "alpine/git:v2.47.1"

@object_type
class FeatureBranch:
    """Feature Branch module for GitHub development"""
    github_token: Annotated[Secret, Doc("GitHub Token")] = field(default=None)
    branch_name: str
    is_fork: bool = False
    branch: Annotated[Directory, Doc("A git repo")] = field(default=dag.directory())
    @classmethod
    async def create(
        cls,
        token: Annotated[Secret, Doc("The Github Token to use")],
        upstream: Annotated[str, Doc("The upstream repository to branch from")],
        branch_name: Annotated[str, Doc("The name of the branch to create")],
        base_ref: Annotated[str, Doc("base ref to branch off of")] | None,
        fork_name: Annotated[str, Doc("The name to give the created forked repo")] | None,
        fork: Annotated[bool, Doc("Should the upstream repo be forked")] = False
    ) -> Self:
        """Creates a feature branch from an upstream repository"""
        # Set fork=True if we have a fork_name
        if fork_name != None:
            fork = True

        branch = dag.git(upstream).head().tree()
        if base_ref is not None:
            branch = dag.git(upstream).ref(base_ref).tree()
        self = cls(github_token = token, is_fork = fork, branch_name = branch_name, branch = branch)

        # Create the fork if specified
        if fork:
            self = await self.fork(upstream, fork_name)
        # Create the branch
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
        changes: Annotated[Directory, Doc("The file changes to apply to the feature branch")],
        keep_git: Annotated[bool, Doc("Keep any .git directory in the changes")] = False
    ) -> Self:
        """Apply a directory of changes to the branch"""
        if keep_git == False:
            changes = changes.without_directory(".git")
        self.branch = self.branch.with_directory(".", changes)
        return self

    @function
    async def diff(
        self
    ) -> str:
        """Returns the diff of the branch"""
        return await (
            self.env()
            .with_exec([
                "git",
                "diff",
            ])
            .stdout()
        )

    @function
    async def push(
        self,
        title: Annotated[str, Doc("The title of the commit")],
    ) -> str:
        """Pushes the branch to the remote"""
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
            ]).stdout()
        )

    @function
    async def pull_request(
        self,
        title: Annotated[str, Doc("The title of the pull request")],
        body: Annotated[str, Doc("The body of the pull request")]
    ) -> str:
        """Creates a pull request on the branch with the provided title and body"""
        origin = await self.get_remote_url("origin")
        upstream = origin
        head = f"{self.branch_name}"
        if self.is_fork:
            upstream = await self.get_remote_url("upstream")
            head = f"{origin.split('/')[-2]}:{self.branch_name}"

        # Push the branch
        await self.push(title)

        # Create the PR
        return await (
            self.env()
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
                head,
            ])
            .stdout()
        )

    @function
    async def get_remote_url(
        self,
        remote: Annotated[str, Doc("The remote name to find the URL for")]
    ) -> str:
        """Returns a remotes full url"""
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
        """Returns a container with the necessary environment for git and gh"""
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
            .with_env_variable("CACHE_BUSTER", str(int(time.time())))
        )

    @function
    async def fork(
        self,
        upstream: Annotated[str, Doc("Upstream URL to fork")],
        fork_name: Annotated[str, Doc("Optional name to give the fork")] | None
    ) -> Self:
        """Sets the branch to a fork of a Github repository"""
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
