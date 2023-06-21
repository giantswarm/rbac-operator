import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
from decorators import retry

LOGGER = logging.getLogger(__name__)

TRIGGER_CROSSPLANE_CLUSTER_ROLE_NAME = "crossplane-edit"
EXPECTED_CLUSTER_ROLE_BINDING_NAME = (
    f"rbac-op-{TRIGGER_CROSSPLANE_CLUSTER_ROLE_NAME}-to-customer-admin"
)


@pytest.mark.smoke
class TestCrossplaneControllerExternalResources:
    kube_client: pykube.HTTPClient

    def init(self, kube_cluster: Cluster):
        if kube_cluster.kube_client is None:
            raise Exception("kube_client should be set")
        self.kube_client = kube_cluster.kube_client

    @pytest.mark.smoke
    def test_crossplane_controller_external_resources(self, kube_cluster: Cluster):
        self.init(kube_cluster)

        crossplane_cluster_role = self.create_clusterrole()
        self.check_created()

        self.delete_clusterrole(crossplane_cluster_role)
        self.check_deleted()

    def create_clusterrole(self) -> pykube.ClusterRole:
        LOGGER.info("Creating crossplane cluster role")
        clusterrole = pykube.ClusterRole(
            self.kube_client,
            {"metadata": {"name": TRIGGER_CROSSPLANE_CLUSTER_ROLE_NAME}},
        )
        clusterrole.create()
        LOGGER.info("Created crossplane cluster role")

        return clusterrole

    @retry()
    def check_created(self):
        LOGGER.info("Checking for expected cluster role binding")
        # raises if not found
        pykube.ClusterRoleBinding.objects(self.kube_client).get(
            name=EXPECTED_CLUSTER_ROLE_BINDING_NAME
        )
        LOGGER.info("Found expected cluster role binding")

    def delete_clusterrole(self, crossplane_cluster_role: pykube.ClusterRole):
        LOGGER.info("Deleting crossplane cluster role")
        crossplane_cluster_role.delete()
        LOGGER.info("Deleted crossplane cluster role")

    @retry()
    def check_deleted(self):
        try:
            pykube.ClusterRoleBinding.objects(self.kube_client).get(
                name=EXPECTED_CLUSTER_ROLE_BINDING_NAME
            )
            raise Exception(
                f"Cluster role binding {EXPECTED_CLUSTER_ROLE_BINDING_NAME} still exists"
            )
        except pykube.exceptions.ObjectDoesNotExist:
            LOGGER.info(
                f"Cluster role binding {EXPECTED_CLUSTER_ROLE_BINDING_NAME} deleted"
            )
