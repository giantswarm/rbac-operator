from typing import List
import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
from pytest_helm_charts.k8s.deployment import wait_for_deployments_to_run
import pytest_helm_charts.k8s.namespace as pytest_namespace


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


@pytest.mark.smoke
@pytest.mark.flaky(reruns=5, reruns_delay=10)
def test_rbac_controller_external_resources(kube_cluster: Cluster):
    LOGGER.info("Creating org and cluster namespaces")
    cluster_namespace_name = "r9b5q"
    org_name = "test"
    org_namespace_name = f"org-{org_name}"

    expected_cluster_role_binding_names = [
        f"organization-organization-{org_name}-read",
        f"releases-organization-{org_name}-read",
    ]

    expected_cluster_role_names = [
        "read-releases",
        f"organization-{org_name}-read",
    ]

    # expected_cluster_role_names = [
    #     f"organization-{cluster_namespace_name}-read",

    kube_client = kube_cluster.kube_client
    if kube_client is None:
        raise Exception("kube_client should be set")

    LOGGER.info("Creating org and cluster namespaces")
    pytest_namespace.ensure_namespace_exists(
        kube_client,
        org_namespace_name,
        extra_metadata={"labels": {"giantswarm.io/organization": org_namespace_name}},
    )

    pytest_namespace.ensure_namespace_exists(
        kube_client,
        cluster_namespace_name,
        extra_metadata={
            "labels": {
                "giantswarm.io/organization": org_namespace_name,
                "giantswarm.io/cluster": cluster_namespace_name,
            }
        },
    )

    # raises if not found
    for expected_cluster_role_name in expected_cluster_role_binding_names:
        pykube.ClusterRoleBinding.objects(kube_client).get(
            name=expected_cluster_role_name
        )
    for expected_cluster_role_name in expected_cluster_role_names:
        pykube.ClusterRole.objects(kube_client).get(name=expected_cluster_role_name)
