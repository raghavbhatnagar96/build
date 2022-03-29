// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrunttlcleanup

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
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

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "buildrun-ttl-cleanup-controller")
	return add(ctx, mgr, NewReconciler(c, mgr), c.Controllers.BuildRun.MaxConcurrentReconciles)
}

func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}

	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles

	}

	c, err := controller.New("buildrun-ttl-cleanup-controller", mgr, options)
	if err != nil {
		return err
	}

	predBuildRun := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Check condition here
			o := e.Object.(*buildv1alpha1.BuildRun)
			condition := o.Status.GetCondition(buildv1alpha1.Succeeded)
			if condition != nil && (condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionTrue) {
				if o.Spec.Retention != nil &&
					(o.Spec.Retention.TTLAfterFailed != nil || o.Spec.Retention.TTLAfterSucceeded != nil) {
					return true
				} else if (o.Status.BuildSpec != nil) && (o.Status.BuildSpec.Retention != nil) &&
					(o.Status.BuildSpec.Retention.TTLAfterFailed != nil || o.Status.BuildSpec.Retention.TTLAfterSucceeded != nil) {
					return true
				}
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*buildv1alpha1.BuildRun)
			o := e.ObjectOld.(*buildv1alpha1.BuildRun)
			oldCondition := o.Status.GetCondition(buildv1alpha1.Succeeded)
			newCondition := n.Status.GetCondition(buildv1alpha1.Succeeded)
			if oldCondition != nil && newCondition != nil {
				if (oldCondition.Status == corev1.ConditionUnknown) &&
					(newCondition.Status == corev1.ConditionFalse || newCondition.Status == corev1.ConditionTrue) {
					if n.Spec.Retention != nil &&
						(n.Spec.Retention.TTLAfterFailed != nil || n.Spec.Retention.TTLAfterSucceeded != nil) {
						return true
					} else if n.Status.BuildSpec != nil && n.Status.BuildSpec.Retention != nil &&
						(n.Status.BuildSpec.Retention.TTLAfterFailed != nil || n.Status.BuildSpec.Retention.TTLAfterSucceeded != nil) {
						return true
					}
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}
	// Watch for changes to primary resource BuildRun
	if err = c.Watch(&source.Kind{Type: &buildv1alpha1.BuildRun{}}, &handler.EnqueueRequestForObject{}, predBuildRun); err != nil {
		return err
	}
	return nil
}
