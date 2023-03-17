package defaultnamespacetest

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/fake"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"
)

type ClientsetWithResources struct {
	*clientgofake.Clientset
	discovery *FakeDiscoveryWithResources
}

func (c *ClientsetWithResources) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

func (c *ClientsetWithResources) WithResources(resources ...metav1.APIResource) *ClientsetWithResources {
	var resourceLists []*metav1.APIResourceList

	for _, resource := range resources {
		groupVersion := fmt.Sprintf("%s/%s", resource.Group, resource.Version)
		var groupResourceList *metav1.APIResourceList
		for _, resourceList := range resourceLists {
			if resourceList.GroupVersion == groupVersion {
				groupResourceList = resourceList
				break
			}
		}
		if groupResourceList == nil {
			groupResourceList = &metav1.APIResourceList{
				GroupVersion: groupVersion,
				APIResources: []metav1.APIResource{},
			}
			resourceLists = append(resourceLists, groupResourceList)
		}
		groupResourceList.APIResources = append(groupResourceList.APIResources, resource)
	}

	c.discovery.Resources = resourceLists
	return c
}

func NewClientSet(objects ...runtime.Object) *ClientsetWithResources {
	cs := &ClientsetWithResources{Clientset: clientgofake.NewSimpleClientset(objects...)}
	cs.discovery = &FakeDiscoveryWithResources{FakeDiscovery: fake.FakeDiscovery{Fake: &cs.Fake}}
	return cs
}

type FakeDiscoveryWithResources struct {
	fake.FakeDiscovery
}

func (c *FakeDiscoveryWithResources) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	sgs, err := c.ServerGroups()
	if err != nil {
		return nil, err
	}
	var preferredVersions []metav1.GroupVersionForDiscovery
	for i := range sgs.Groups {
		preferredVersions = append(preferredVersions, sgs.Groups[i].PreferredVersion)
	}

	var preferredResources []*metav1.APIResourceList
	for _, resource := range c.Resources {
		for _, preferredVersion := range preferredVersions {
			if preferredVersion.GroupVersion == resource.GroupVersion {
				preferredResources = append(preferredResources, resource)
				break
			}
		}
	}

	action := testing2.ActionImpl{
		Verb:     "get",
		Resource: schema.GroupVersionResource{Resource: "resource"},
	}
	_, err = c.Invokes(action, nil)
	if err != nil {
		return preferredResources, err
	}
	return preferredResources, nil
}
