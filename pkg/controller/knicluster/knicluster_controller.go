package knicluster

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	kniv1alpha1 "github.com/mhrivnak/kni-operator/pkg/apis/kni/v1alpha1"
	osconfigv1 "github.com/openshift/api/config/v1"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_knicluster")

const (
	// FinalizerName is the finalizer value used on non-owned resources
	FinalizerName          = "knicluster.kni.openshift.com"
	KNIClusterNameEnv      = "KNI_CLUSTER_NAME"
	KNIClusterNameDefault  = "kni-cluster"
	KNIClusterNamespaceEnv = "KNI_CLUSTER_NAMESPACE"
)

// Add creates a new KNICluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKNICluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knicluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KNICluster
	err = c.Watch(&source.Kind{Type: &kniv1alpha1.KNICluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch secondary resources
	for _, resource := range []runtime.Object{
		&olmv1.OperatorGroup{},
		&olm.CatalogSource{},
		&olm.Subscription{},
	} {
		err = c.Watch(&source.Kind{Type: resource}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &kniv1alpha1.KNICluster{},
		})
		if err != nil {
			return err
		}
	}

	kni, err := GetKNINamespacedName()
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &osconfigv1.ClusterVersion{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(
			// always enqueued the same KNICluster object, since there should be only one
			func(a handler.MapObject) []reconcile.Request {
				return []reconcile.Request{
					{NamespacedName: kni},
				}
			}),
	})

	return err
}

// blank assignment to verify that ReconcileKNICluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKNICluster{}

// ReconcileKNICluster reconciles a KNICluster object
type ReconcileKNICluster struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileKNICluster) ensureOperatorGroup(instance *kniv1alpha1.KNICluster, reqLogger logr.Logger) error {
	// ensure OperatorGroup exists
	operatorGroup := newOperatorGroup(instance.Namespace)
	if err := controllerutil.SetControllerReference(instance, operatorGroup, r.scheme); err != nil {
		return err
	}

	// Check if this OperatorGroup already exists
	found := &olmv1.OperatorGroup{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: operatorGroup.Name, Namespace: operatorGroup.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new OperatorGroup", "OperatorGroup.Namespace", operatorGroup.Namespace, "OperatorGroup.Name", operatorGroup.Name)
		err = r.client.Create(context.TODO(), operatorGroup)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	// already exists - don't requeue
	reqLogger.Info("OperatorGroup already exists", "OperatorGroup.Namespace", found.Namespace, "OperatorGroup.Name", found.Name)

	return nil
}

func (r *ReconcileKNICluster) ensureSubscription(instance *kniv1alpha1.KNICluster, reqLogger logr.Logger) error {
	// ensure Subscription exists
	subscription := newSubscription(instance.Namespace)
	if err := controllerutil.SetControllerReference(instance, subscription, r.scheme); err != nil {
		return err
	}

	// Check if this Subscription already exists
	found := &olm.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: subscription.Name, Namespace: subscription.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Subscription", "Subscription.Namespace", subscription.Namespace, "Subscription.Name", subscription.Name)
		err = r.client.Create(context.TODO(), subscription)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	// already exists - don't requeue
	reqLogger.Info("Subscription already exists", "Subscription.Namespace", found.Namespace, "Subscription.Name", found.Name)

	return nil
}

func (r *ReconcileKNICluster) ensureCatalogSource(instance *kniv1alpha1.KNICluster, reqLogger logr.Logger) error {
	cvs := osconfigv1.ClusterVersionList{}
	err := r.client.List(context.TODO(), &client.ListOptions{}, &cvs)
	if err != nil {
		return err
	}
	if len(cvs.Items) != 1 {
		return fmt.Errorf("Expected 1 ClusterVersion, found %d", len(cvs.Items))
	}
	cv := &cvs.Items[0]

	// ensure CatalogSource exists
	catalogsource := newCatalogSource(cv.Spec.DesiredUpdate.Version)

	// Check if this CatalogSource already exists
	found := &olm.CatalogSource{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: catalogsource.Name, Namespace: catalogsource.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new CatalogSource", "CatalogSource.Namespace", catalogsource.Namespace, "CatalogSource.Name", catalogsource.Name)
		err = r.client.Create(context.TODO(), catalogsource)
		if err != nil {
			return err
		}

		// created successfully - don't requeue
		return nil
	} else if err != nil {
		return err
	}

	// already exists - don't requeue
	reqLogger.Info("CatalogSource already exists", "CatalogSource.Namespace", found.Namespace, "CatalogSource.Name", found.Name)

	// update the image if necessary
	if found.Spec.Image != catalogsource.Spec.Image {
		reqLogger.Info("Updating the CatalogSource image", "CatalogSource.Namespace", found.Namespace, "CatalogSource.Name", found.Name)
		found.Spec.Image = catalogsource.Spec.Image
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func (r *ReconcileKNICluster) ensureCatalogSourceDeleted() error {
	cs := newCatalogSource("latest")
	err := r.client.Delete(context.TODO(), cs)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// Reconcile reads that state of the cluster for a KNICluster object and makes changes based on the state read
// and what is in the KNICluster.Spec
func (r *ReconcileKNICluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KNICluster")

	// Fetch the KNICluster instance
	instance := &kniv1alpha1.KNICluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Not Found")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, FinalizerName)
			return reconcile.Result{}, r.client.Update(context.TODO(), instance)
		}
	} else {
		if containsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			err = r.ensureCatalogSourceDeleted()
			if err != nil {
				return reconcile.Result{}, err
			}

			// remove finalizer
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, FinalizerName)
			return reconcile.Result{}, r.client.Update(context.TODO(), instance)
		}
	}

	for _, f := range []func(*kniv1alpha1.KNICluster, logr.Logger) error{
		r.ensureOperatorGroup,
		r.ensureCatalogSource,
		r.ensureSubscription,
	} {
		err = f(instance, reqLogger)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func newCatalogSource(version string) *olm.CatalogSource {
	return &olm.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-catalog",
			Namespace: "olm",
		},
		Spec: olm.CatalogSourceSpec{
			SourceType: olm.SourceTypeGrpc,
			// TODO get this from the Status and ensure the update is complete
			Image:       fmt.Sprintf("quay.io/mhrivnak/demo-operator-registry:%s", version),
			Publisher:   "kni.openshift.com",
			DisplayName: "KNI Operators",
		},
	}
}

func newSubscription(namespace string) *olm.Subscription {
	return &olm.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kni",
			Namespace: namespace,
		},
		Spec: &olm.SubscriptionSpec{
			Channel:                "singlenamespace-alpha",
			Package:                "etcd",
			CatalogSource:          "demo-catalog",
			CatalogSourceNamespace: "olm",
		},
	}
}

func newOperatorGroup(namespace string) *olmv1.OperatorGroup {
	return &olmv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kni",
			Namespace: namespace,
		},
		Spec: olmv1.OperatorGroupSpec{
			TargetNamespaces: []string{namespace},
		},
	}
}

func GetKNINamespacedName() (types.NamespacedName, error) {
	kni := types.NamespacedName{
		Name: KNIClusterNameDefault,
	}

	// get name
	if name, ok := os.LookupEnv(KNIClusterNameEnv); ok {
		kni.Name = name
	}

	// get namespace
	if namespace, ok := os.LookupEnv(KNIClusterNamespaceEnv); ok {
		kni.Namespace = namespace
	} else {
		return kni, fmt.Errorf("%s unset or empty", KNIClusterNamespaceEnv)
	}
	return kni, nil
}
