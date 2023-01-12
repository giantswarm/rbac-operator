from typing import List

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
from pytest_helm_charts.k8s.deployment import wait_for_deployments_to_run


timeout: int = 60

deployment_names = ["rbac-operator"]
namespace_name = "default"


@pytest.mark.smoke
def test_api_working(kube_cluster: Cluster) -> None:
    assert kube_cluster.kube_client is not None
    assert len(pykube.Node.objects(kube_cluster.kube_client)) >= 1


@pytest.fixture(scope="module")
def app_deployments(kube_cluster: Cluster) -> List[pykube.Deployment]:
    if kube_cluster.kube_client is None:
        raise Exception("kube_client is None")
    print(pykube.Deployment.objects(kube_cluster.kube_client))
    deployments = wait_for_deployments_to_run(
        kube_cluster.kube_client,
        deployment_names,
        namespace_name,
        timeout,
    )
    return deployments


@pytest.mark.smoke
@pytest.mark.upgrade
@pytest.mark.flaky(reruns=5, reruns_delay=10)
def test_pods_available(
    kube_cluster: Cluster, app_deployments: List[pykube.Deployment]
):
    for d in app_deployments:
        assert int(d.obj["status"]["readyReplicas"]) > 0