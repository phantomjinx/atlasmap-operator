/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
)

// AtlasMapReconciler reconciles a AtlasMap object
type AtlasMapReconciler struct {
	client       client.Client
	scheme       *runtime.Scheme
	config       *rest.Config
	configClient *configv1client.Clientset
}

var actions []action

//+kubebuilder:rbac:groups=atlasmap.io.atlasmap.io,resources=atlasmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=atlasmap.io.atlasmap.io,resources=atlasmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=atlasmap.io.atlasmap.io,resources=atlasmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AtlasMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *AtlasMapReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AtlasMap")

	// Fetch the AtlasMap instance
	instance := &v1alpha1.AtlasMap{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			instance.ObjectMeta = metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			}

			isOpenShift, _ := util.IsOpenShift(r.config)
			if isOpenShift && util.IsOpenShift43Plus(ctx, r.config) {
				//Handle removal of cluster-scope object.
				return r.removeConsoleLink(instance)
			}

			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, a := range actions {
		reqLogger.Info("Running action: " + a.getName())
		if err := a.handle(ctx, instance); err != nil {
			if errors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			reqLogger.Error(err, "Error running action: "+a.getName())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AtlasMapReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	// Create a new controller
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AtlasMap{}).
		Owns(&appsv1.Deployment{})

	isOpenShift, err := util.IsOpenShift(mgr.GetConfig())
	if err != nil {
		return err
	}

	if isOpenShift {
		builder.Owns(&routev1.Route{})
	} else {
		builder.Owns(&netv1.Ingress{})
	}

	actions = newOperatorActions(ctx, log, mgr)

	return builder.Complete(r)
}

func (r *AtlasMapReconciler) removeConsoleLink(atlasMap *v1alpha1.AtlasMap) (request reconcile.Result, err error) {
	consoleLinkName := util.ConsoleLinkName(atlasMap)
	consoleLink := &consolev1.ConsoleLink{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	} else {
		err = r.client.Delete(context.TODO(), consoleLink)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
