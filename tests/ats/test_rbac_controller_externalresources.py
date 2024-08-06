import logging
from typing import Tuple

import pykube
import pytest
from pytest_helm_charts.fixtures import Cluster
import pytest_helm_charts.k8s.namespace as pytest_namespace
from pytest_helm_charts.k8s.wait_for import wait_for_objects_to_appear, wait_for_objects_to_disappear

LOGGER = logging.getLogger(__name__)

CLUSTER_NAMESPACE_NAME = "r9b5q"
ORG_NAME = "test"
ORG_NAMESPACE_NAME = f"org-{ORG_NAME}"

EXPECTED_CLUSTER_ROLE_BINDING_NAMES = [
    f"organization-organization-{ORG_NAME}-read",
    f"releases-organization-{ORG_NAME}-read",
]

EXPECTED_CLUSTER_ROLE_NAMES = [
    "read-releases",
    f"organization-{ORG_NAME}-read",
]


@pytest.mark.smoke
class TestRBACControllerExternalResources:

    @pytest.mark.smoke
    @pytest.mark.flaky(reruns=3, reruns_delay=10)
    def test_rbac_controller_external_resources(self, kube_cluster: Cluster):
        org_namespace, cluster_namespace = self.create_namespaces(kube_cluster.kube_client)
        self.check_created(kube_cluster.kube_client)
        self.delete_namespaces(kube_cluster.kube_client, cluster_namespace, org_namespace)
        self.check_deleted(kube_cluster.kube_client)

    def create_namespaces(self, kube_client) -> Tuple[pykube.Namespace, pykube.Namespace]:
        LOGGER.info("Creating org and cluster namespaces")
        org_namespace = pytest_namespace.create_namespace(
            kube_client,
            ORG_NAMESPACE_NAME,
            labels={"giantswarm.io/organization": ORG_NAMESPACE_NAME}
        )
        cluster_namespace = pytest_namespace.create_namespace(
            kube_client,
            CLUSTER_NAMESPACE_NAME,
            labels={
                "giantswarm.io/organization": ORG_NAMESPACE_NAME,
                "giantswarm.io/cluster": CLUSTER_NAMESPACE_NAME,
            }
        )
        LOGGER.info("Created org and cluster namespaces")
        return org_namespace, cluster_namespace

    def check_created(self, kube_client):
        LOGGER.info("Checking for expected cluster role bindings and roles")
        wait_for_objects_to_appear(
            kube_client,
            EXPECTED_CLUSTER_ROLE_BINDING_NAMES,
            pykube.ClusterRoleBinding,
            timeout_seconds=60,
        )
        wait_for_objects_to_appear(
            kube_client,
            EXPECTED_CLUSTER_ROLE_NAMES,
            pykube.ClusterRole,
            timeout_seconds=60,
        )
        LOGGER.info("Found expected cluster role bindings and roles")

    def delete_namespaces(self, kube_client, cluster_namespace: pykube.Namespace, org_namespace: pykube.Namespace):
        LOGGER.info("Deleting org and cluster namespaces")
        pytest_namespace.delete_namespace(kube_client, cluster_namespace.name)
        pytest_namespace.delete_namespace(kube_client, org_namespace.name)
        LOGGER.info("Deleted org and cluster namespaces")

    def check_deleted(self, kube_client):
        LOGGER.info("Checking for deletion of cluster role bindings and roles")
        wait_for_objects_to_disappear(
            kube_client,
            EXPECTED_CLUSTER_ROLE_BINDING_NAMES,
            pykube.ClusterRoleBinding,
            timeout_seconds=120,
        )
        wait_for_objects_to_disappear(
            kube_client,
            EXPECTED_CLUSTER_ROLE_NAMES,
            pykube.ClusterRole,
            timeout_seconds=120,
        )
        LOGGER.info("Confirmed deletion of cluster role bindings and roles")