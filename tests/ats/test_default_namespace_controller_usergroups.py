import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
from decorators import retry


LOGGER = logging.getLogger(__name__)

EXPECTED_CLUSTER_ROLE_BINDING_NAMES = [
    "read-all-customer-group",
    "write-all-giantswarm-group",
]


@pytest.mark.smoke
class TestDefaultNameSpaceControllerUserGroups:
    kube_client: pykube.HTTPClient

    def init(self, kube_cluster: Cluster):
        if kube_cluster.kube_client is None:
            raise Exception("kube_client should be set")
        self.kube_client = kube_cluster.kube_client

    @pytest.mark.smoke
    def test_rbac_controller_external_resources(self, kube_cluster: Cluster):
        self.init(kube_cluster)

        self.check_created()

    @retry()
    def check_created(self):
        LOGGER.info("Checking for expected cluster role bindings and roles")
        # raises if not found
        for expected_cluster_role_name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES:
            pykube.ClusterRoleBinding.objects(self.kube_client).get(
                name=expected_cluster_role_name
            )
        LOGGER.info("Found expected cluster role bindings and roles")
