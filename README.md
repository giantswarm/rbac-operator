[![CircleCI](https://circleci.com/gh/giantswarm/rbac-operator.svg?&style=shield&&circle-token=373dcae33aecb47a0a53c51105e9381dff5b0b88)](https://circleci.com/gh/giantswarm/rbac-operator) [![Docker Repository on Quay](https://quay.io/repository/giantswarm/rbac-operator/status "Docker Repository on Quay")](https://quay.io/repository/giantswarm/rbac-operator)

# rbac-operator

The rbac-operator is an in-cluster agent, that manages roles and rolebingings
for tenant cluster namespaces inside the Giant Swarm control-plane Kubernetes cluster.

## Getting Project

Clone the git repository: https://github.com/giantswarm/rbac-operator.git

### How to build

Build it using the standard `go build` command.

```
go build github.com/giantswarm/rbac-operator
```

## Contact

- Mailing list: [giantswarm](https://groups.google.com/forum/!forum/giantswarm)
- IRC: #[giantswarm](irc://irc.freenode.org:6667/#giantswarm) on freenode.org
- Bugs: [issues](https://github.com/giantswarm/rbac-operator/issues)


## License

rbac-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for
details.
