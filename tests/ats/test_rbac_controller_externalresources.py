import logging
import time
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
        
        try:
            self.check_created()
        except Exception as e:
            LOGGER.error(f"Failed to create resources: {str(e)}")
            self.log_existing_resources()
            raise

        self.delete_namespaces(cluster_namespace, org_namespace)
        
        try:
            self.check_deleted()
        except Exception as e:
            LOGGER.error(f"Failed to delete resources: {str(e)}")
            self.log_existing_resources()
            raise

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

    @retry(max_retries=10, delay=5)
    def check_created(self):
        LOGGER.info("Checking for expected cluster role bindings and roles")
        missing_resources = []

        for expected_name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES:
            try:
                pykube.ClusterRoleBinding.objects(self.kube_client).get(name=expected_name)
                LOGGER.info(f"Found ClusterRoleBinding: {expected_name}")
            except pykube.exceptions.ObjectDoesNotExist:
                missing_resources.append(f"ClusterRoleBinding: {expected_name}")

        for expected_name in EXPECTED_CLUSTER_ROLE_NAMES:
            try:
                pykube.ClusterRole.objects(self.kube_client).get(name=expected_name)
                LOGGER.info(f"Found ClusterRole: {expected_name}")
            except pykube.exceptions.ObjectDoesNotExist:
                missing_resources.append(f"ClusterRole: {expected_name}")

        if missing_resources:
            raise Exception(f"The following resources are missing: {', '.join(missing_resources)}")

        LOGGER.info("Found all expected cluster role bindings and roles")

    def delete_namespaces(
        self, cluster_namespace: pykube.Namespace, org_namespace: pykube.Namespace
    ):
        LOGGER.info("Deleting org and cluster namespaces")
        org_namespace.delete()
        cluster_namespace.delete()
        LOGGER.info("Deleted org and cluster namespaces")

    @retry(max_retries=20, delay=5)
    def check_deleted(self):
        LOGGER.info("Checking if resources have been deleted")
        existing_resources = []

        for expected_name in EXPECTED_CLUSTER_ROLE_BINDING_NAMES:
            try:
                pykube.ClusterRoleBinding.objects(self.kube_client).get(name=expected_name)
                existing_resources.append(f"ClusterRoleBinding: {expected_name}")
            except pykube.exceptions.ObjectDoesNotExist:
                LOGGER.info(f"ClusterRoleBinding deleted: {expected_name}")

        for expected_name in EXPECTED_CLUSTER_ROLE_NAMES:
            try:
                pykube.ClusterRole.objects(self.kube_client).get(name=expected_name)
                existing_resources.append(f"ClusterRole: {expected_name}")
            except pykube.exceptions.ObjectDoesNotExist:
                LOGGER.info(f"ClusterRole deleted: {expected_name}")

        if existing_resources:
            raise Exception(f"The following resources still exist: {', '.join(existing_resources)}")

        LOGGER.info("All expected cluster role bindings and roles have been deleted")

    def log_existing_resources(self):
        LOGGER.info("Logging existing ClusterRoleBindings and ClusterRoles")
        try:
            crbs = pykube.ClusterRoleBinding.objects(self.kube_client)
            LOGGER.info(f"Existing ClusterRoleBindings: {[crb.name for crb in crbs]}")
        except Exception as e:
            LOGGER.error(f"Error listing ClusterRoleBindings: {str(e)}")

        try:
            crs = pykube.ClusterRole.objects(self.kube_client)
            LOGGER.info(f"Existing ClusterRoles: {[cr.name for cr in crs]}")
        except Exception as e:
            LOGGER.error(f"Error listing ClusterRoles: {str(e)}")