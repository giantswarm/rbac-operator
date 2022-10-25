# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- Extend readonly ClusterRole to include pods/log so customers can debug failures on the management clusters.

## [0.30.2] - 2022-08-11

### Changed

- Adjusted detection of changes in roles and role bindings to prevent unnecessary updates during reconciliation

## [0.30.1] - 2022-08-03

### Changed

- In case an organization can not be fetched, try fetching it via the annotation `ui.giantswarm.io/original-organization-name`.

## [0.30.0] - 2022-08-02

### Added

- Reconcile individual clusterRoleBinding for each org namespace to grant and revoke automation access to silences.

## [0.29.0] - 2022-05-30

### Added

- Add write-silences for flux automation account in every org-namespace

## [0.28.1] - 2022-05-18

### Fixed

- Remove `app-operator-*-chart` ClusterRole and ClusterRoleBinding for old app-operator versions.

## [0.28.0] - 2022-05-16

### Added

- Add new roles and cluster roles to app-operators when a cluster namespace is created and clean them up when the cluster namespace is destroyed

### Removed

- Remove `app-operator-*` ClusterRole and ClusterRoleBinding for old app-operator versions.
- Remove `write-flux-resources-customer-sa` ClusterRoleBinding.

## [0.27.0] - 2022-04-28

### Added

- Create `flux-crd-controller` and `flux-namespace-reconciler` RoleBindings in cluster namespace.

### Changed

- Update CI (architext-orb)

## [0.26.0] - 2022-04-01

### Removed

- Remove `orgpermissions` controller as it is deprecated and all its resources have been cleaned up.
## [0.25.0] - 2022-03-31

### Changed

- Use `k8smetadata` for labels and annotations instead of `apiextensions`.
- Use `organization-operator` to import `Organization` CRD.
- Bump `giantswarm/k8sclient` to `v7.0.1`.
- Bump `giantswarm/operatorkit` to `v7.0.1`.
- Bump k8s dependencies to `v0.20.15`.
- Bump `controller-runtime` to `v0.8.3`.
- Selector moved from watcher to individual handlers for rbac controller.

## [0.24.0] - 2022-03-30

### Added

- Add `externalresources` resource that binds `organization-$org-read` role for any subject with org-namespace access.
- Deprecate `orgpermissions` controller as it is replaced by the above.

## [0.23.0] - 2022-03-02

### Added

- Prevent rbac-controller `fluxauth` and `externalresources` resources from reconciling cluster namespaces
- Dynamically bind `read-in-cluster-ns` clusterRole if `read-all` clusterRole is bound in an org-namespace
- Dynamically bind `write-in-cluster-ns` clusterRole if `cluster-admin` clusterRole is bound in an org-namespace


### Changed

- Renamed role `read-cluster-apps-in-cluster-ns` to `read-in-cluster-ns`
- Renamed role `write-cluster-apps-in-cluster-ns` to `write-in-cluster-ns`
- Renamed role binding `read-cluster-app` to `read-in-cluster-ns`
- Renamed role binding `write-cluster-apps` to `write-in-cluster-ns`

## [0.22.1] - 2022-02-21

- Update some role descriptions.
- Enhance log messages in the bootstrapping part, remove unneeded messages.

## [0.22.0] - 2022-02-18

### Added

- Add `externalresources` resource that binds `read-default-catalogs` and `read-releases` roles for any subject with org-namespace access.
- Add creation of `read-default-catalogs` Role.
- Add creation of `read-releases` ClusterRole.
- Improve logging for the `orgpermissions`, `clusternamespace`, and `rbac` controllers.

## [0.21.0] - 2022-02-16

### Added

- Add cluster-namespace controller which ensures that RBAC resources to access resources in cluster namespaces can be granted to those with access to the clusters organization
- Add bootstrapping for the `read-cluster-apps` and `write-cluster-apps` clusterRoles.
- Add update option for `orgReadClusterRoleBinding` resource.

### Changed

- The `write_all_group` configuration key is now optional.

## [0.20.0] - 2022-02-07

### Added

- Create RBAC for customer-facing Flux to access organization namespaces.
- Add `automation` ServiceAccount to organization namespaces with permissions to handle Flux resources in that namespace by default.

## [0.19.1] - 2022-02-02

### Fixed

- Add missing `imagePullSecret`.

## [0.19.0] - 2021-12-17

### Added

- Add user-friendly descriptions to created `ClusterRole` resources, via annotations using the `giantswarm.io/notes` key.

### Changed

- Change the CI build process to use architect-orb.
- Adapt code to mitigate warnings occurring for common code checks.
- Modify log messages for updating ClusterRoles.
- Require Go v1.16.

## [0.18.4] - 2021-12-10

### Fixed

- Fix missing selfLink issue by updating to operatorkit@v4.3.1.

## [0.18.3] - 2021-11-22

### Added

- Added missing bootstrap for Silence and ClientCert roles for automation SA

## [0.18.2] - 2021-11-22

### Added

- Provide access to the customer automation SA for managing workload cluster silences.

## [0.18.1] - 2021-11-17

## Added

- Set the `giantswarm.io/managed-by` label on the `cluster-admin` RBAC ClusterRole.

## [0.18.0] - 2021-11-11

### Added

- Provide access to the customer automation SA for managing workload cluster client certificates.

## [0.17.0] - 2021-11-04

### Added

- Provide access to the customer automation SA for managing flux resources.
- Provide access to the customer automation SA for managing cluster-specific resources.
- Provide access to the customer automation SA for managing node pool-specific resources.

## [0.16.0] - 2021-10-07

### Added

- Provide access for customer automation SA `Organization` CR management.

## [0.15.0] - 2021-05-25

### Changed

- Prepare Helm values to use config CR.
- Update architect-orb to v3.0.0.

## [0.14.0] - 2021-04-26

### Added

- Provide customer admin group access to manage `Organization` CRs.

## [0.13.0] - 2021-04-26

### Changed

- Update bootstrap resources on restart.

## [0.12.0] - 2021-03-30

### Changed

- Install all the `rbac-operator` resources by default.

## [0.11.0] - 2021-03-25

### Changed

- Label default ClusterRoles, which needs to be displayed in Happa.

### Deleted

- Remove selector in `orgpermissions` controller.

## [0.10.0] - 2021-03-23

### Changed

- Update `read-all` ClusterRole on every bootstrap.
- Extend `rbac-operator` service account ClusterRole permissions to namespaces.

## [0.9.0] - 2021-03-22

### Changed

- Move management of static resources from Helm into code.
- Remove `view-all` related roles/bindings.
- Bind customer admin group to `cluster-admin` cluster role in target organization namespace.

## [0.8.0] - 2020-11-19

### Added

- `clusterrolebinding` for GiantSwarm staff cluster-admin access.

## [0.7.0] - 2020-10-21

### Added

- Update Roles when their Rules are not up to date.

### Fixed

- Update `RoleBindings` only when necessary.

## [0.6.0] - 2020-09-24

### Changed

- Updated Kubernetes dependencies to v1.18.9 and operatorkit to v2.0.1.

### Added

- Add monitoring labels.

## [0.5.0] - 2020-08-14

### Changed

- Updated backward incompatible Kubernetes dependencies to v1.18.5.

## [0.4.6] - 2020-08-13

### Changed

- Update operatorkit to v1.2.0 and k8sclient to v3.1.2.

## [0.4.5] - 2020-07-30

### Fixed

- Fix `roleRef` in `RoleBinding`/`tenant-admin`.

## [0.4.4] - 2020-07-30

### Fixed

- Fix `roleRef` in `ClusterRoleBinding`/`tenant-admin-view-all`.

## [0.4.3] - 2020-07-29

### Added

- Add github actions for release automation.

### Changed

- Update helm chart to current standard
- Install `serviceaccount` in all installations.

## [0.4.2] - 2020-05-03

### Changed

- Change `rbac` controller label selector to match organization namespaces as well.

## [0.4.1]

- Fix `namespacelabeler` controller label selector.
- Fix `role` name reference in OIDC group and service accounts `rolebinding`.

## [0.4.0]

### Changed

- Push tags to *aliyun* repository.
- Move `rbac` controller code into `rbac` package.
- Add `namespacelabeler` controller, which labels legacy namespaces.
- Add `automation` service account in `global` namespace, which has admin access to all the tenant namespaces.

## [0.3.3] - 2020-05-13

### Changed

- Reconcile `rolebinding` subject group changes properly.
- Fix bug with binding role to the `view-all` read role instead of `tenant-admin` write role.


## [0.3.2]- 2020-04-23

### Changed

- Use Release.Revision in annotation for Helm 3 compatibility.


## [0.3.0]- 2020-04-06

### Added

- Tenant admin role *tenant-admin-manage-rbac* to manage `serviceaccounts`, `roles`, `clusterroles`, `rolebindings` and `clusterrolebindings`.
- Add tenant admin full access to `global` and `default` namespaces.

## [0.2.0]- 2020-03-13

### Changed

- Make rbac-operator optional for installation without OIDC.


## [0.1.0]- 2020-03-13

### Added

- Read-only role for customer access into Control Plane.

[Unreleased]: https://github.com/giantswarm/rbac-operator/compare/v0.30.2...HEAD
[0.30.2]: https://github.com/giantswarm/rbac-operator/compare/v0.30.1...v0.30.2
[0.30.1]: https://github.com/giantswarm/rbac-operator/compare/v0.30.0...v0.30.1
[0.30.0]: https://github.com/giantswarm/rbac-operator/compare/v0.29.0...v0.30.0
[0.29.0]: https://github.com/giantswarm/rbac-operator/compare/v0.28.1...v0.29.0
[0.28.1]: https://github.com/giantswarm/rbac-operator/compare/v0.28.0...v0.28.1
[0.28.0]: https://github.com/giantswarm/rbac-operator/compare/v0.27.0...v0.28.0
[0.27.0]: https://github.com/giantswarm/rbac-operator/compare/v0.26.0...v0.27.0
[0.26.0]: https://github.com/giantswarm/rbac-operator/compare/v0.25.0...v0.26.0
[0.25.0]: https://github.com/giantswarm/rbac-operator/compare/v0.24.0...v0.25.0
[0.24.0]: https://github.com/giantswarm/rbac-operator/compare/v0.23.0...v0.24.0
[0.23.0]: https://github.com/giantswarm/rbac-operator/compare/v0.22.1...v0.23.0
[0.22.1]: https://github.com/giantswarm/rbac-operator/compare/v0.22.0...v0.22.1
[0.22.0]: https://github.com/giantswarm/rbac-operator/compare/v0.21.0...v0.22.0
[0.21.0]: https://github.com/giantswarm/rbac-operator/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/giantswarm/rbac-operator/compare/v0.19.1...v0.20.0
[0.19.1]: https://github.com/giantswarm/rbac-operator/compare/v0.19.0...v0.19.1
[0.19.0]: https://github.com/giantswarm/rbac-operator/compare/v0.18.4...v0.19.0
[0.18.4]: https://github.com/giantswarm/rbac-operator/compare/v0.18.3...v0.18.4
[0.18.3]: https://github.com/giantswarm/rbac-operator/compare/v0.18.2...v0.18.3
[0.18.2]: https://github.com/giantswarm/rbac-operator/compare/v0.18.1...v0.18.2
[0.18.1]: https://github.com/giantswarm/rbac-operator/compare/v0.18.0...v0.18.1
[0.18.0]: https://github.com/giantswarm/rbac-operator/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/giantswarm/rbac-operator/compare/v0.16.0...v0.17.0
[0.16.0]: https://github.com/giantswarm/rbac-operator/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/giantswarm/rbac-operator/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/giantswarm/rbac-operator/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/giantswarm/rbac-operator/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/giantswarm/rbac-operator/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/giantswarm/rbac-operator/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/giantswarm/rbac-operator/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/giantswarm/rbac-operator/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/giantswarm/rbac-operator/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/giantswarm/rbac-operator/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/giantswarm/rbac-operator/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/giantswarm/rbac-operator/compare/v0.4.6...v0.5.0
[0.4.6]: https://github.com/giantswarm/rbac-operator/compare/v0.4.5...v0.4.6
[0.4.5]: https://github.com/giantswarm/rbac-operator/compare/v0.4.4...v0.4.5
[0.4.4]: https://github.com/giantswarm/rbac-operator/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/giantswarm/rbac-operator/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.4.2
[0.4.1]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.4.1
[0.4.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.4.0
[0.3.3]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.3.3
[0.3.2]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.3.2
[0.3.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.3.0
[0.2.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.2.0
[0.1.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.1.0
