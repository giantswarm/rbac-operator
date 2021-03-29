# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/rbac-operator/compare/v0.11.0...HEAD
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
