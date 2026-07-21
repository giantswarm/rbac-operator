import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
import pytest_helm_charts.k8s.namespace as pytest_namespace

from decorators import retry

LOGGER = logging.getLogger(__name__)

# The Role and RoleBinding live in the `giantswarm` namespace and are required
# for the App to HelmRelease migration. A single Role/RoleBinding is created and
# the `automation` ServiceAccount of every organization namespace is aggregated
# into the RoleBinding subjects.
GIANTSWARM_NAMESPACE = "giantswarm"
PATCH_CHARTS_NAME = "patch-charts"
AUTOMATION_SA_NAME = "automation"

ORG_A = "pctesta"
ORG_B = "pctestb"
ORG_A_NAMESPACE = f"org-{ORG_A}"
ORG_B_NAMESPACE = f"org-{ORG_B}"


@pytest.mark.smoke
class TestRBACControllerPatchCharts:
    kube_client: pykube.HTTPClient

    def init(self, kube_cluster: Cluster):
        if kube_cluster.kube_client is None:
            raise Exception("kube_client should be set")
        self.kube_client = kube_cluster.kube_client

    @pytest.mark.smoke
    @pytest.mark.flaky(reruns=3, reruns_delay=10)
    def test_rbac_controller_patch_charts(self, kube_cluster: Cluster):
        self.init(kube_cluster)

        self.ensure_giantswarm_namespace()
        org_a, org_b = self.create_org_namespaces()

        # both orgs' automation ServiceAccounts should be aggregated into the
        # shared patch-charts RoleBinding, and the Role should exist.
        self.check_role_created()
        self.check_subjects({ORG_A_NAMESPACE, ORG_B_NAMESPACE})

        # deleting one org namespace must drop only its subject, while the Role
        # and RoleBinding (with the remaining org) stay in place.
        self.delete_namespace(org_b)
        self.check_subjects({ORG_A_NAMESPACE}, absent={ORG_B_NAMESPACE})

        self.delete_namespace(org_a)

    def ensure_giantswarm_namespace(self):
        LOGGER.info("Ensuring %s namespace exists", GIANTSWARM_NAMESPACE)
        pytest_namespace.ensure_namespace_exists(self.kube_client, GIANTSWARM_NAMESPACE)

    def create_org_namespaces(self):
        LOGGER.info("Creating org namespaces")
        org_a = self.create_org_namespace(ORG_A_NAMESPACE)
        org_b = self.create_org_namespace(ORG_B_NAMESPACE)
        LOGGER.info("Created org namespaces")
        return org_a, org_b

    def create_org_namespace(self, name: str) -> pykube.Namespace:
        namespace, _ = pytest_namespace.ensure_namespace_exists(
            self.kube_client,
            name,
            extra_metadata={"labels": {"giantswarm.io/organization": name}},
        )
        return namespace

    @retry()
    def check_role_created(self):
        LOGGER.info("Checking for the %s Role", PATCH_CHARTS_NAME)
        role = pykube.Role.objects(
            self.kube_client, namespace=GIANTSWARM_NAMESPACE
        ).get(name=PATCH_CHARTS_NAME)

        chart_rules = [
            rule
            for rule in role.obj.get("rules", [])
            if "charts" in rule.get("resources", [])
            and "application.giantswarm.io" in rule.get("apiGroups", [])
        ]
        assert chart_rules, f"{PATCH_CHARTS_NAME} Role has no rule for charts"

        verbs = set(chart_rules[0].get("verbs", []))
        assert {"list", "get", "patch"} <= verbs, (
            f"unexpected verbs on {PATCH_CHARTS_NAME} Role: {verbs}"
        )
        LOGGER.info("Found the %s Role with the expected rules", PATCH_CHARTS_NAME)

    @retry()
    def check_subjects(self, present: set, absent: set = frozenset()):
        LOGGER.info(
            "Checking %s RoleBinding subjects (present=%s, absent=%s)",
            PATCH_CHARTS_NAME,
            present,
            absent,
        )
        namespaces = self.automation_sa_namespaces()

        missing = present - namespaces
        assert not missing, f"expected automation SAs for {missing} in subjects"

        unexpected = absent & namespaces
        assert not unexpected, f"did not expect automation SAs for {unexpected} in subjects"
        LOGGER.info("RoleBinding subjects are as expected")

    def automation_sa_namespaces(self) -> set:
        role_binding = pykube.RoleBinding.objects(
            self.kube_client, namespace=GIANTSWARM_NAMESPACE
        ).get(name=PATCH_CHARTS_NAME)

        subjects = role_binding.obj.get("subjects") or []
        return {
            subject["namespace"]
            for subject in subjects
            if subject.get("kind") == "ServiceAccount"
            and subject.get("name") == AUTOMATION_SA_NAME
        }

    def delete_namespace(self, namespace: pykube.Namespace):
        LOGGER.info("Deleting namespace %s", namespace.name)
        namespace.delete()
        LOGGER.info("Deleted namespace %s", namespace.name)
