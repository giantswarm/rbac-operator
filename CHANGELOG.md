# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Reconcile `rolebinding` subject group changes properly.
- Fix bug with binding role to the `view-all` read role instead of `tenant-admin` write role.

### Deleted

- Delete `versionbundle` from project.


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

[Unreleased]: https://github.com/giantswarm/rbac-operator/compare/v0.3.2...HEAD

[0.3.2]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.3.2

[0.3.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.3.0

[0.2.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.2.0

[0.1.0]: https://github.com/giantswarm/rbac-operator/releases/tag/v0.1.0
