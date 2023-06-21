import logging
import time
from typing import Tuple

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
import pytest_helm_charts.k8s.namespace as pytest_namespace


LOGGER = logging.getLogger(__name__)

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


def retry(max_retries=5, delay=10):
    def decorator(func):
        def wrapper(*args, **kwargs):
            retries = 0
            while retries < max_retries:
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    retries += 1
                    time.sleep(delay)
                    LOGGER.debug(f"Retrying {func.__name__} due to error: {e}")
            raise Exception("Max retries exceeded")

        return wrapper

    return decorator


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
            org_namespace_name,
            extra_metadata={
                "labels": {"giantswarm.io/organization": org_namespace_name}
            },
        )

        cluster_namespace, _ = pytest_namespace.ensure_namespace_exists(
            self.kube_client,
            cluster_namespace_name,
            extra_metadata={
                "labels": {
                    "giantswarm.io/organization": org_namespace_name,
                    "giantswarm.io/cluster": cluster_namespace_name,
                }
            },
        )
        LOGGER.info("Created org and cluster namespaces")

        return org_namespace, cluster_namespace

    @retry()
    def check_created(self):
        LOGGER.info("Checking for expected cluster role bindings and roles")
        # raises if not found
        for expected_cluster_role_name in expected_cluster_role_binding_names:
            pykube.ClusterRoleBinding.objects(self.kube_client).get(
                name=expected_cluster_role_name
            )
        for expected_cluster_role_name in expected_cluster_role_names:
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

    @retry()
    def check_deleted(self):
        try:
            for expected_cluster_role_name in expected_cluster_role_binding_names:
                pykube.ClusterRoleBinding.objects(self.kube_client).get(
                    name=expected_cluster_role_name
                )
            for expected_cluster_role_name in expected_cluster_role_names:
                pykube.ClusterRole.objects(self.kube_client).get(
                    name=expected_cluster_role_name
                )
            raise Exception("Cluster role bindings and roles still exist")
        except pykube.exceptions.ObjectDoesNotExist:
            LOGGER.info("Cluster role bindings and roles deleted")
