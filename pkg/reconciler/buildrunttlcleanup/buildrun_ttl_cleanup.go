// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrunttlcleanup

import (
	"context"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuildRun reconciles a BuildRun object

type ReconcileBuildRun struct {
	/* This client, initialized using mgr.Client() above, is a split client
	   that reads objects from the cache and writes to the apiserver */
	config *config.Config
	client client.Client
}

func NewReconciler(c *config.Config, mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBuildRun{
		config: c,
		client: mgr.GetClient(),
	}
}

/* Reconciles TTL if it exists. It returns time left to reach TTL and returns
the error if we are unable to delete the buildrun and the error is not that
the buildrun was not found */
func (r *ReconcileBuildRun) reconcileTTL(ctx context.Context, ttlDuration *v1.Duration, br *buildv1alpha1.BuildRun, request reconcile.Request) (*time.Duration, error) {
	if ttlDuration == nil {
		return nil, nil
	}
	if br.Status.CompletionTime.Add(ttlDuration.Duration).Before(time.Now()) {
		ctxlog.Info(ctx, "Deleting buildrun as ttl has been reached.", namespace, request.Namespace, name, request.Name)
		err := r.client.Delete(ctx, br, &client.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				ctxlog.Debug(ctx, "Error deleting buildRun.", namespace, request.Namespace, name, br.Name, "error", err)
				return nil, err
			}
			return nil, nil
		}
	} else {
		timeLeft := time.Until(br.Status.CompletionTime.Add(ttlDuration.Duration))
		return &timeLeft, nil
	}
	return nil, nil
}

/* Reconciler makes sure the buildrun adheres to its ttl retention field and deletes it
   once the ttl limit is hit */
func (r *ReconcileBuildRun) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.if
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "Start reconciling Buildrun-ttl", namespace, request.Namespace, name, request.Name)

	br := &buildv1alpha1.BuildRun{}
	err := r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, br)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Debug(ctx, "Finish reconciling buildrun-ttl. Buildrun was not found", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var TTLFailed *v1.Duration
	var TTLSucceeded *v1.Duration
	if br.Spec.Retention == nil && br.Status.BuildSpec != nil && br.Status.BuildSpec.Retention != nil {
		TTLFailed = br.Status.BuildSpec.Retention.TTLAfterFailed
		TTLSucceeded = br.Status.BuildSpec.Retention.TTLAfterSucceeded
	} else if br.Spec.Retention != nil {
		TTLFailed = br.Spec.Retention.TTLAfterFailed
		TTLSucceeded = br.Spec.Retention.TTLAfterSucceeded
		if TTLFailed == nil && br.Status.BuildSpec != nil && br.Status.BuildSpec.Retention != nil {
			TTLFailed = br.Status.BuildSpec.Retention.TTLAfterFailed
		}
		if TTLSucceeded == nil && br.Status.BuildSpec != nil && br.Status.BuildSpec.Retention != nil {
			TTLSucceeded = br.Status.BuildSpec.Retention.TTLAfterSucceeded
		}
	}

	condition := br.Status.GetCondition(buildv1alpha1.Succeeded)

	/* In case ttl has been reached, delete the buildrun, if not,
	   calculate the remaining time and requeue the buildrun */
	switch condition.Status {

	case corev1.ConditionTrue:
		timeLeft, err := r.reconcileTTL(ctx, TTLSucceeded, br, request)
		if err != nil {
			return reconcile.Result{}, err
		}
		if timeLeft != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: *timeLeft}, nil
		}
		return reconcile.Result{}, nil

	case corev1.ConditionFalse:
		timeLeft, err := r.reconcileTTL(ctx, TTLFailed, br, request)
		if err != nil {
			return reconcile.Result{}, err
		}
		if timeLeft != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: *timeLeft}, nil
		}
	}
	ctxlog.Debug(ctx, "Finishing reconciling request from a BuildRun event", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}
