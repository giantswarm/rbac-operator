from typing import List
import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
from pytest_helm_charts.k8s.deployment import wait_for_deployments_to_run


LOGGER = logging.getLogger(__name__)

TIMEOUT: int = 60

DEPLOYMENT_NAMES = ["rbac-operator"]
NAMESPACE_NAME = "default"


@pytest.mark.smoke
def test_api_working(kube_cluster: Cluster) -> None:
    assert kube_cluster.kube_client is not None
    assert len(pykube.Node.objects(kube_cluster.kube_client)) >= 1


@pytest.fixture(scope="module")
def app_deployments(kube_cluster: Cluster) -> List[pykube.Deployment]:
    if kube_cluster.kube_client is None:
        raise Exception("kube_client is None")
    LOGGER.info("Installing crd")
    kube_cluster.kubectl("apply", filename="https://raw.githubusercontent.com/giantswarm/rbac-operator/main/config/crd/auth.giantswarm.io_rolebindingtemplates.yaml", output_format="json")
    kube_cluster.kubectl("apply", filename="https://raw.githubusercontent.com/giantswarm/organization-operator/refs/heads/main/config/crd/bases/security.giantswarm.io_organizations.yaml", output_format="json")
    LOGGER.info("Installed crd")
    deployments = wait_for_deployments_to_run(
        kube_cluster.kube_client,
        DEPLOYMENT_NAMES,
        NAMESPACE_NAME,
        TIMEOUT,
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
