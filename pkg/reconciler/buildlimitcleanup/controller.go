// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildlimitcleanup

import (
	"context"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	namespace string = "namespace"
	name      string = "name"
)

// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "build-limit-cleanup-controller")
	return add(ctx, mgr, NewReconciler(c, mgr), c.Controllers.Build.MaxConcurrentReconciles)
}
func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}

	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles
	}

	// Create a new controller
	c, err := controller.New("build-limit-cleanup-controller", mgr, options)
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*buildv1alpha1.Build)
			return o.Spec.Retention != nil && (o.Spec.Retention.FailedLimit != nil || o.Spec.Retention.SucceededLimit != nil)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*buildv1alpha1.Build)
			o := e.ObjectOld.(*buildv1alpha1.Build)

			/* Check to see if there are new retention parameters or whether the
			limit values have decreased */
			if o.Spec.Retention == nil && n.Spec.Retention != nil {
				if n.Spec.Retention.FailedLimit != nil || n.Spec.Retention.SucceededLimit != nil {
					return true
				}
			} else if n.Spec.Retention != nil && o.Spec.Retention != nil {
				if n.Spec.Retention.FailedLimit != nil && o.Spec.Retention.FailedLimit == nil {
					return true
				} else if n.Spec.Retention.SucceededLimit != nil && o.Spec.Retention.SucceededLimit == nil {
					return true
				} else if n.Spec.Retention.FailedLimit != nil && o.Spec.Retention.FailedLimit != nil && int(*n.Spec.Retention.FailedLimit) < int(*o.Spec.Retention.FailedLimit) {
					return true
				} else if n.Spec.Retention.SucceededLimit != nil && o.Spec.Retention.SucceededLimit != nil && int(*n.Spec.Retention.SucceededLimit) < int(*o.Spec.Retention.SucceededLimit) {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	predBuildRun := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Never reconcile in case of create buildrun event
			return false
		},
		// Reconcile the build the related buildrun has just completed
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*buildv1alpha1.BuildRun)
			o := e.ObjectOld.(*buildv1alpha1.BuildRun)
			// check if Buildrun is related to a build
			if n.Spec.BuildRef.Name == "" {
				return false
			}
			oldCondition := o.Status.GetCondition(buildv1alpha1.Succeeded)
			newCondition := n.Status.GetCondition(buildv1alpha1.Succeeded)
			if oldCondition != nil && newCondition != nil {
				if (oldCondition.Status == corev1.ConditionUnknown) &&
					(newCondition.Status == corev1.ConditionFalse || newCondition.Status == corev1.ConditionTrue) {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	// Watch for changes to primary resource Build
	if err = c.Watch(&source.Kind{Type: &build.Build{}}, &handler.EnqueueRequestForObject{}, pred); err != nil {
		return err
	}

	// Watch for changes to resource BuildRun
	return c.Watch(&source.Kind{Type: &buildv1alpha1.BuildRun{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		buildRun := o.(*buildv1alpha1.BuildRun)

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      buildRun.Spec.BuildRef.Name,
					Namespace: buildRun.Namespace,
				},
			},
		}
	}), predBuildRun)
}
