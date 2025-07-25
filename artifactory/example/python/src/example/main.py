import time
import dagger
from dagger import dag, function, object_type


@object_type
class Example:
    @function
    async def example__upload_artifact_with_evidence(
        self,
        instance_url: str,
        access_token: dagger.Secret,
    ) -> str:
        """Upload an artifact with evidence"""

        # Construct the artifactory module
        client = dag.artifactory(access_token, instance_url)

        # Upload an artifact
        artifact = dag.http("https://github.com/dagger/dagger/releases/download/v0.18.13/dagger_v0.18.13_linux_amd64.tar.gz")
        artifact_path = "generic-repo/dagger_v0.18.13_linux_amd64.tar.gz"
        upload_output = await client.upload(artifact, artifact_path)

        # Wait a moment for upload to process
        time.sleep(1)

        # Upload evidence for the artifact
        predicate = dag.current_module().source().file("./test_predicate.json")
        predicate_type = "https://in-toto.io/Statement/v1"
        key = dag.current_module().source().file("./private.pem")
        key_alias = "OtherKey"
        trace_url = await dag.cloud().trace_url()
        create_evidence_output = await client.create_evidence(predicate, predicate_type, key, key_alias=key_alias, subject_repo_path=artifact_path, trace_url=trace_url)

        return upload_output + "\n" + create_evidence_output
