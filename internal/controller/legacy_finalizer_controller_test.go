package controller_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/giantswarm/rbac-operator/api/v1alpha1"
	"github.com/giantswarm/rbac-operator/internal/controller"
)

const otherFinalizer = "example.com/keep-me"

func TestLegacyFinalizerReconciler(t *testing.T) {
	now := metav1.Now()

	testCases := []struct {
		name string
		obj  client.Object
		// expectedFinalizers is nil when the object is expected to be gone.
		expectedFinalizers []string
	}{
		{
			name: "removes the rbac-controller finalizer from an unlabelled namespace",
			obj: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:       "kube-system",
				Finalizers: []string{"operatorkit.giantswarm.io/rbac-operator-rbac-controller", otherFinalizer},
			}},
			expectedFinalizers: []string{otherFinalizer},
		},
		{
			name: "removes all namespace finalizers of the old controllers",
			obj: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "org-acme",
				Finalizers: []string{
					"operatorkit.giantswarm.io/rbac-operator-rbac-controller",
					"operatorkit.giantswarm.io/rbac-operator-crossplane-namespace-controller",
					"operatorkit.giantswarm.io/rbac-operator-cluster-namespace-controller",
					"operatorkit.giantswarm.io/rbac-operator-default-namespace-controller",
				},
			}},
			expectedFinalizers: []string{},
		},
		{
			name: "unblocks a namespace being deleted",
			obj: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:              "terminating",
				DeletionTimestamp: &now,
				Finalizers:        []string{"operatorkit.giantswarm.io/rbac-operator-rbac-controller"},
			}},
			expectedFinalizers: nil,
		},
		{
			name: "removes the crossplane-controller finalizer from any ClusterRole",
			obj: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name:       "view",
				Finalizers: []string{"operatorkit.giantswarm.io/rbac-operator-crossplane-controller", otherFinalizer},
			}},
			expectedFinalizers: []string{otherFinalizer},
		},
		{
			name: "removes the rolebindingtemplate-controller finalizer",
			obj: &v1alpha1.RoleBindingTemplate{ObjectMeta: metav1.ObjectMeta{
				Name:       "template",
				Finalizers: []string{"operatorkit.giantswarm.io/rbac-operator-rolebindingtemplate-controller"},
			}},
			expectedFinalizers: []string{},
		},
		{
			name: "keeps finalizers that belong to another kind",
			obj: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:       "default",
				Finalizers: []string{"operatorkit.giantswarm.io/rbac-operator-crossplane-controller"},
			}},
			expectedFinalizers: []string{"operatorkit.giantswarm.io/rbac-operator-crossplane-controller"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			c := newLegacyFinalizerClient(t, tc.obj)
			r := legacyFinalizerReconcilerFor(t, c, tc.obj)

			before := tc.obj.DeepCopyObject().(client.Object)
			if err := c.Get(ctx, client.ObjectKeyFromObject(tc.obj), before); err != nil {
				t.Fatalf("getting object: %v", err)
			}

			if _, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tc.obj)}); err != nil {
				t.Fatalf("reconcile: %v", err)
			}

			after := tc.obj.DeepCopyObject().(client.Object)
			err := c.Get(ctx, client.ObjectKeyFromObject(tc.obj), after)
			if tc.expectedFinalizers == nil {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("expected object to be deleted once its finalizers are gone, got err=%v finalizers=%v", err, after.GetFinalizers())
				}
				return
			}
			if err != nil {
				t.Fatalf("getting object: %v", err)
			}
			if !slices.Equal(tc.expectedFinalizers, after.GetFinalizers()) {
				t.Fatalf("unexpected finalizers - expected %v, received %v", tc.expectedFinalizers, after.GetFinalizers())
			}
			// Nothing to remove means no write.
			if len(tc.expectedFinalizers) == len(before.GetFinalizers()) && after.GetResourceVersion() != before.GetResourceVersion() {
				t.Fatalf("expected no update, resourceVersion changed from %s to %s", before.GetResourceVersion(), after.GetResourceVersion())
			}
		})
	}
}

func TestLegacyFinalizerReconciler_MissingObject(t *testing.T) {
	c := newLegacyFinalizerClient(t)
	for _, r := range controller.LegacyFinalizerReconcilers(c) {
		if _, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKey{Name: "gone"}}); err != nil {
			t.Fatalf("%s: expected no error for a missing object, got %v", r.Name, err)
		}
	}
}

func newLegacyFinalizerClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// legacyFinalizerReconcilerFor returns the reconciler that handles obj's kind.
func legacyFinalizerReconcilerFor(t *testing.T, c client.Client, obj client.Object) *controller.LegacyFinalizerReconciler {
	t.Helper()
	for _, r := range controller.LegacyFinalizerReconcilers(c) {
		if reflect.TypeOf(r.NewObject()) == reflect.TypeOf(obj) {
			return r
		}
	}
	t.Fatalf("no legacy finalizer reconciler for %T", obj)
	return nil
}
