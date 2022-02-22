/*


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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	basev1 "jiffykube.io/jiffykube/api/v1"
	"jiffykube.io/jiffykube/operations"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const appFinalizerName = "apps.base.jiffykube.io/finzliazer"

// +kubebuilder:rbac:groups=base.jiffykube.io,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=base.jiffykube.io,resources=apps/status,verbs=get;update;patch

func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("app", req.NamespacedName)

	// get app
	var app basev1.App
	if err := r.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// add finalizer to serve as pre-delete hook
	controllerutil.AddFinalizer(&app, appFinalizerName)
	if err := r.Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println("APP", app.Name, app.Namespace)

	// delete app
	if !app.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, &app)
		return ctrl.Result{}, err
	}
	// create/update app
	if err := r.upsert(ctx, &app); err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&basev1.App{}).
		Complete(r)
}

func (r *AppReconciler) upsert(ctx context.Context, app *basev1.App) error {
	fmt.Println("UPSERT")
	errChan := make(chan error)
	go func() {
		errChan <- operations.UpsertDeployment(ctx, r.Client, app)
		errChan <- operations.UpsertService(ctx, r.Client, app)
		errChan <- operations.UpsertIngress(ctx, r.Client, app)
	}()
	for i := 0; i < 3; i++ {
		err := <-errChan
		fmt.Println("Error upserting", err) // TODO handle
	}
	return nil
}

func (r *AppReconciler) delete(ctx context.Context, app *basev1.App) error {
	fmt.Println("DELETE")
	errChan := make(chan error)
	go func() {
		errChan <- operations.DeleteIngress(ctx, r.Client, app)
		errChan <- operations.DeleteService(ctx, r.Client, app)
		errChan <- operations.DeleteDeployment(ctx, r.Client, app)
	}()

	for i := 0; i < 3; i++ {
		err := <-errChan
		fmt.Println("Error deleting", err) // TODO handle
	}

	// remove app finalizer
	controllerutil.RemoveFinalizer(app, appFinalizerName)
	if err := r.Update(ctx, app); err != nil {
		return err
	}
	return nil
}
