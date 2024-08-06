import logging
import time
from typing import Tuple

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
import pytest_helm_charts.k8s.namespace as pytest_namespace

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
        org_namespace, _ = pytest_namespace.ensure_namespace_exists(
            kube_client,
            ORG_NAMESPACE_NAME,
            extra_metadata={
                "labels": {"giantswarm.io/organization": ORG_NAMESPACE_NAME}
            },
        )
        cluster_namespace, _ = pytest_namespace.ensure_namespace_exists(
            kube_client,
            CLUSTER_NAMESPACE_NAME,
            extra_metadata={
                "labels": {
                    "giantswarm.io/organization": ORG_NAMESPACE_NAME,
                    "giantswarm.io/cluster": CLUSTER_NAMESPACE_NAME,
                }
            },
        )
        LOGGER.info("Created org and cluster namespaces")
        return org_namespace, cluster_namespace

    def check_created(self, kube_client):
        LOGGER.info("Checking for expected cluster role bindings and roles")
        max_retries = 10
        for _ in range(max_retries):
            try:
                for name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES:
                    pykube.ClusterRoleBinding.objects(kube_client).get(name=name)
                for name in EXPECTED_CLUSTER_ROLE_NAMES:
                    pykube.ClusterRole.objects(kube_client).get(name=name)
                LOGGER.info("Found expected cluster role bindings and roles")
                return
            except pykube.exceptions.ObjectDoesNotExist:
                time.sleep(6)
        raise TimeoutError("Timed out waiting for cluster role bindings and roles to be created")

    def delete_namespaces(self, kube_client, cluster_namespace: pykube.Namespace, org_namespace: pykube.Namespace):
        LOGGER.info("Deleting org and cluster namespaces")
        cluster_namespace.delete()
        org_namespace.delete()
        LOGGER.info("Deleted org and cluster namespaces")

    def check_deleted(self, kube_client):
        LOGGER.info("Checking for deletion of cluster role bindings and roles")
        max_retries = 20
        for _ in range(max_retries):
            all_deleted = True
            for name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES + EXPECTED_CLUSTER_ROLE_NAMES:
                try:
                    pykube.ClusterRoleBinding.objects(kube_client).get(name=name)
                    all_deleted = False
                    break
                except pykube.exceptions.ObjectDoesNotExist:
                    pass
            if all_deleted:
                LOGGER.info("Confirmed deletion of cluster role bindings and roles")
                return
            time.sleep(6)
        raise TimeoutError("Timed out waiting for cluster role bindings and roles to be deleted")