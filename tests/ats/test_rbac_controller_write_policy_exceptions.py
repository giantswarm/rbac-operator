import logging

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster
import pytest_helm_charts.k8s.namespace as pytest_namespace

from decorators import retry

LOGGER = logging.getLogger(__name__)

# The Role and RoleBinding live in every namespace in which PolicyExceptions are
# managed. Per namespace a single Role/RoleBinding is created and the
# `automation` ServiceAccount of every organization namespace is aggregated into
# the RoleBinding subjects.
WRITE_POLICY_EXCEPTIONS_NAMESPACES = ["kube-system", "giantswarm", "policy-exceptions"]
WRITE_POLICY_EXCEPTIONS_NAME = "write-policy-exceptions"
AUTOMATION_SA_NAME = "automation"

ORG_A = "wpetesta"
ORG_B = "wpetestb"
ORG_C = "wpetestc"
ORG_A_NAMESPACE = f"org-{ORG_A}"
ORG_B_NAMESPACE = f"org-{ORG_B}"
ORG_C_NAMESPACE = f"org-{ORG_C}"


@pytest.mark.smoke
class TestRBACControllerWritePolicyExceptions:
    kube_client: pykube.HTTPClient

    def init(self, kube_cluster: Cluster):
        if kube_cluster.kube_client is None:
            raise Exception("kube_client should be set")
        self.kube_client = kube_cluster.kube_client

    @pytest.mark.smoke
    @pytest.mark.flaky(reruns=3, reruns_delay=10)
    def test_rbac_controller_write_policy_exceptions(self, kube_cluster: Cluster):
        self.init(kube_cluster)

        # the target namespaces have to exist before the org namespaces are
        # created, as the operator skips the ones that are absent.
        self.ensure_target_namespaces()
        org_a, org_b = self.create_org_namespaces()

        # both orgs' automation ServiceAccounts should be aggregated into the
        # shared write-policy-exceptions RoleBinding of every target namespace,
        # and the Roles should exist.
        self.check_roles()
        self.check_subjects({ORG_A_NAMESPACE, ORG_B_NAMESPACE})

        # if a Role's rules are changed out-of-band, the operator must reconcile
        # them back to the desired state. Creating another org namespace triggers
        # that reconcile immediately, rather than relying on the operator's
        # ~5 minute periodic resync to eventually catch it.
        self.drift_role_rules(WRITE_POLICY_EXCEPTIONS_NAMESPACES[0])
        org_c = self.create_org_namespace(ORG_C_NAMESPACE)
        self.check_roles()
        self.check_subjects({ORG_A_NAMESPACE, ORG_B_NAMESPACE, ORG_C_NAMESPACE})

        # deleting one org namespace must drop only its subject, while the Roles
        # and RoleBindings (with the remaining orgs) stay in place.
        self.delete_namespace(org_b)
        self.check_subjects({ORG_A_NAMESPACE, ORG_C_NAMESPACE}, absent={ORG_B_NAMESPACE})

        self.delete_namespace(org_a)
        self.delete_namespace(org_c)

    def ensure_target_namespaces(self):
        for namespace in WRITE_POLICY_EXCEPTIONS_NAMESPACES:
            LOGGER.info("Ensuring %s namespace exists", namespace)
            pytest_namespace.ensure_namespace_exists(self.kube_client, namespace)

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
    def check_roles(self):
        for namespace in WRITE_POLICY_EXCEPTIONS_NAMESPACES:
            LOGGER.info(
                "Checking for the %s Role in namespace %s",
                WRITE_POLICY_EXCEPTIONS_NAME,
                namespace,
            )
            role = pykube.Role.objects(self.kube_client, namespace=namespace).get(
                name=WRITE_POLICY_EXCEPTIONS_NAME
            )

            policy_exception_rules = [
                rule
                for rule in role.obj.get("rules", [])
                if "policyexceptions" in rule.get("resources", [])
                and "kyverno.io" in rule.get("apiGroups", [])
            ]
            assert policy_exception_rules, (
                f"{WRITE_POLICY_EXCEPTIONS_NAME} Role in namespace {namespace} "
                "has no rule for policyexceptions"
            )

            verbs = set(policy_exception_rules[0].get("verbs", []))
            assert verbs == {"*"}, (
                f"unexpected verbs on {WRITE_POLICY_EXCEPTIONS_NAME} Role in "
                f"namespace {namespace}: {verbs}"
            )
            LOGGER.info(
                "Found the %s Role with the expected rules in namespace %s",
                WRITE_POLICY_EXCEPTIONS_NAME,
                namespace,
            )

    @retry()
    def check_subjects(self, present: set, absent: set = frozenset()):
        for namespace in WRITE_POLICY_EXCEPTIONS_NAMESPACES:
            LOGGER.info(
                "Checking %s RoleBinding subjects in namespace %s (present=%s, absent=%s)",
                WRITE_POLICY_EXCEPTIONS_NAME,
                namespace,
                present,
                absent,
            )
            namespaces = self.automation_sa_namespaces(namespace)

            missing = present - namespaces
            assert not missing, (
                f"expected automation SAs for {missing} in subjects of the "
                f"RoleBinding in namespace {namespace}"
            )

            unexpected = absent & namespaces
            assert not unexpected, (
                f"did not expect automation SAs for {unexpected} in subjects of "
                f"the RoleBinding in namespace {namespace}"
            )
            LOGGER.info(
                "RoleBinding subjects in namespace %s are as expected", namespace
            )

    def automation_sa_namespaces(self, namespace: str) -> set:
        role_binding = pykube.RoleBinding.objects(
            self.kube_client, namespace=namespace
        ).get(name=WRITE_POLICY_EXCEPTIONS_NAME)

        subjects = role_binding.obj.get("subjects") or []
        return {
            subject["namespace"]
            for subject in subjects
            if subject.get("kind") == "ServiceAccount"
            and subject.get("name") == AUTOMATION_SA_NAME
        }

    def drift_role_rules(self, namespace: str):
        LOGGER.info(
            "Drifting the %s Role's rules in namespace %s",
            WRITE_POLICY_EXCEPTIONS_NAME,
            namespace,
        )
        role = pykube.Role.objects(self.kube_client, namespace=namespace).get(
            name=WRITE_POLICY_EXCEPTIONS_NAME
        )

        role.obj["rules"] = [
            {
                "apiGroups": ["kyverno.io"],
                "resources": ["policyexceptions"],
                "verbs": ["get"],
            }
        ]
        role.update()
        LOGGER.info(
            "Drifted the %s Role's rules in namespace %s",
            WRITE_POLICY_EXCEPTIONS_NAME,
            namespace,
        )

    def delete_namespace(self, namespace: pykube.Namespace):
        LOGGER.info("Deleting namespace %s", namespace.name)
        namespace.delete()
        LOGGER.info("Deleted namespace %s", namespace.name)
