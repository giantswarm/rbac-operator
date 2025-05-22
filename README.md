# rbac-operator

[![CircleCI](https://circleci.com/gh/giantswarm/rbac-operator.svg?&style=shield&&circle-token=373dcae33aecb47a0a53c51105e9381dff5b0b88)](https://circleci.com/gh/giantswarm/rbac-operator)
[![Docker Repository on Quay](https://quay.io/repository/giantswarm/rbac-operator/status "Docker Repository on Quay")](https://quay.io/repository/giantswarm/rbac-operator)

The rbac-operator is a Kubernetes operator that manages RBAC-related resources in Giant Swarm management clusters. It automates the creation and maintenance of roles, role bindings, and service accounts to ensure proper access control across the platform.

Customers use this operator to simplify access management across their Kubernetes environment, reduce administrative overhead, and maintain consistent security policies while integrating with their existing identity providers.

## Overview

The rbac-operator handles several key aspects of RBAC management:

- Creates and maintains ClusterRoles with specific permissions
- Manages RoleBindings and ClusterRoleBindings for customer and Giant Swarm admin groups
- Creates ServiceAccounts for automation purposes
- Provides access to organization namespaces, cluster namespaces, and their resources
- Supports integration with external identity providers through OIDC
- Manages Flux-related permissions for GitOps workflows

## Architecture

The rbac-operator consists of multiple controllers, each responsible for managing different aspects of RBAC resources:

1. **DefaultNamespace Controller** - Manages roles and role bindings in the default namespace
2. **ClusterNamespace Controller** - Handles RBAC resources for cluster namespaces
3. **RBAC Controller** - Creates and maintains organization-specific RBAC resources
4. **Crossplane Controller** - Manages permissions for Crossplane resources
5. **RoleBindingTemplate Controller** - Supports templating of role bindings across multiple namespaces

## Features

### Customer and admin access groups

The operator manages access for different user groups:

- Customer admin groups (full access to organization resources)
- Customer reader groups (read-only access to resources)
- Giant Swarm admin groups (platform-wide administrative access)

### Permission scopes

The operator manages permissions at different scopes:

- Cluster-wide permissions (ClusterRoles and ClusterRoleBindings)
- Organization namespace permissions
- Cluster namespace permissions
- Default namespace permissions

## Configuration

The rbac-operator can be configured using the following settings:

### Helm values

```yaml
oidc:
  customer:
    write_all_groups:                                           # Customer groups with admin access
      - "customer-idp:giantswarm:Admins"
    read_all_groups:                                            # Customer groups with read-only access
      - "customer-idp:giantswarm:Readers"
  giantswarm:
    write_all_groups:                                           # Giant Swarm admin groups
      - "giantswarm-ad:giantswarm-admins"
```

## Custom resources

The rbac-operator supports custom RoleBindingTemplate resources:

```yaml
apiVersion: auth.giantswarm.io/v1alpha1
kind: RoleBindingTemplate
metadata:
  name: example-template
spec:
  template:
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: example-role
    subjects:
    - kind: ServiceAccount
      name: example-sa
    - kind: Group
      name: example-group
  scopes:
    organizationSelector:
      matchLabels:
        key: value
```

### RoleBindingTemplate

The RoleBindingTemplate is a powerful feature that enables dynamic management of RoleBindings across multiple namespaces. It's particularly useful in multi-tenant environments where consistent access control policies need to be maintained across multiple organization namespaces.

#### Use Cases

- **Automate Access Management**: Define RBAC policies once and have them automatically applied across all relevant namespaces
- **Simplify Operations**: Reduce the manual work required to manage permissions across multiple namespaces
- **Scale Access Control**: Easily manage permissions as the organization grows and new namespaces are created
- **Service Account Management**: Grant consistent permissions to service accounts across multiple namespaces
- **Organization-wide Policies**: Set up and maintain access policies across all organization namespaces
- **Dynamic RBAC Setup**: Automate RBAC configuration for new organizations or clusters

## Development

### Building the operator

Build it using the standard `go build` command.

```bash
go build github.com/giantswarm/rbac-operator
```

## Contact

- Mailing list: [giantswarm](https://groups.google.com/forum/!forum/giantswarm)
- Bugs: [issues](https://github.com/giantswarm/rbac-operator/issues)

## License

rbac-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for
details.
