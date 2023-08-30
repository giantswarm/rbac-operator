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

EXPECTED_ROLE_BINDING_NAMES = [
    "write-all-customer-sa"
]


@pytest.mark.smoke
class TestDefaultNamespaceControllerFluxAuth:
    kube_client: pykube.HTTPClient

    def init(self, kube_cluster: Cluster):
        if kube_cluster.kube_client is None:
            raise Exception("kube_client should be set")
        self.kube_client = kube_cluster.kube_client

    @pytest.mark.smoke
    def test_default_namespace_controller_fluxauth(self, kube_cluster: Cluster):
        self.init(kube_cluster)

        org_namespace, cluster_namespace = self.create_namespaces()
        self.create_organization(kube_cluster)
        self.check_created()

        self.delete_organization()
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
    
    def create_organization(self, kube_cluster: Cluster):
        LOGGER.info("Creating organization")
        kube_cluster.kubectl("apply", filename="test-organization.yaml", output_format="json")
        LOGGER.info("Created organization")
    
    @retry()
    def check_created(self):
        LOGGER.info("Checking for expected role bindings")
        # raises if not found
        for expected_rolebinding_name in EXPECTED_ROLE_BINDING_NAMES:
            pykube.RoleBinding.objects(self.kube_client).get(
                name=expected_rolebinding_name
            )
        LOGGER.info("Found expected role bindings")

    def delete_namespaces(self, cluster_namespace: pykube.Namespace, org_namespace: pykube.Namespace):
        LOGGER.info("Deleting org and cluster namespaces")
        cluster_namespace.delete()
        org_namespace.delete()
        LOGGER.info("Deleted org and cluster namespaces")

    def delete_organization(self, kube_cluster: Cluster):
        LOGGER.info("Deleting organization")
        kube_cluster.kubectl("delete", "organizations.security.giantswarm.io", ORG_NAME, output_format="json")
        LOGGER.info("Deleted organization")

    @retry()
    def check_deleted(self):
        try:
            for expected_rolebinding_name in EXPECTED_ROLE_BINDING_NAMES:
                pykube.RoleBinding.objects(self.kube_client).get(
                    name=expected_rolebinding_name
                )
            raise Exception(
                f"Role binding {expected_rolebinding_name} still exists")
        except pykube.exceptions.ObjectDoesNotExist:
            LOGGER.info(
                f"Role binding {expected_rolebinding_name} deleted")
