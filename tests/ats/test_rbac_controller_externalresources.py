import logging
from typing import Tuple

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
import pytest_helm_charts.k8s.namespace as pytest_namespace
from decorators import retry

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
    kube_client: pykube.HTTPClient

    def init(self, kube_cluster: Cluster):
        if kube_cluster.kube_client is None:
            raise Exception("kube_client should be set")
        self.kube_client = kube_cluster.kube_client

    @pytest.mark.smoke
    def test_rbac_controller_external_resources(self, kube_cluster: Cluster):
        self.init(kube_cluster)

        org_namespace, cluster_namespace = self.create_namespaces()
        self.check_created()

        self.delete_namespaces(cluster_namespace, org_namespace)
        self.check_deleted()

    def create_namespaces(self) -> Tuple[pykube.Namespace, pykube.Namespace]:
        LOGGER.info("Creating org and cluster namespaces")
        org_namespace, _ = pytest_namespace.ensure_namespace_exists(
            self.kube_client,
            ORG_NAMESPACE_NAME,
            extra_metadata={
                "labels": {"giantswarm.io/organization": ORG_NAMESPACE_NAME}
            },
        )

        cluster_namespace, _ = pytest_namespace.ensure_namespace_exists(
            self.kube_client,
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

    @retry(max_retries=10)
    def check_created(self):
        LOGGER.info("Checking for expected cluster role bindings and roles")
        # raises if not found
        for expected_cluster_role_name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES:
            pykube.ClusterRoleBinding.objects(self.kube_client).get(
                name=expected_cluster_role_name
            )
        for expected_cluster_role_name in EXPECTED_CLUSTER_ROLE_NAMES:
            pykube.ClusterRole.objects(self.kube_client).get(
                name=expected_cluster_role_name
            )
        LOGGER.info("Found expected cluster role bindings and roles")

    def delete_namespaces(
        self, cluster_namespace: pykube.Namespace, org_namespace: pykube.Namespace
    ):
        LOGGER.info("Deleting org and cluster namespaces")
        org_namespace.delete()
        cluster_namespace.delete()
        LOGGER.info("Deleted org and cluster namespaces")

    @retry(max_retries=20)
    def check_deleted(self):
        try:
            for expected_cluster_role_name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES:
                pykube.ClusterRoleBinding.objects(self.kube_client).get(
                    name=expected_cluster_role_name
                )
            for expected_cluster_role_name in EXPECTED_CLUSTER_ROLE_NAMES:
                pykube.ClusterRole.objects(self.kube_client).get(
                    name=expected_cluster_role_name
                )
            raise Exception("Cluster role bindings and roles still exist")
        except pykube.exceptions.ObjectDoesNotExist:
            LOGGER.info("Cluster role bindings and roles deleted")
