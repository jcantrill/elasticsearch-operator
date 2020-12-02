package controllers

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/ViaQ/logerr/log"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	loggingv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"github.com/openshift/elasticsearch-operator/internal/k8shandler"
)

// ElasticsearchReconciler reconciles a Elasticsearch object
type ElasticsearchReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Elasticsearch object and makes changes based on the state read
// and what is in the Elasticsearch.Spec
var (
	reconcilePeriod = 30 * time.Second
	// reconcileResult = reconcile.Result{RequeueAfter: reconcilePeriod}
	reconcileResult = ctrl.Result{RequeueAfter: reconcilePeriod}
)

func (r *ElasticsearchReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// Fetch the Elasticsearch instance
	cluster := &loggingv1.Elasticsearch{}

	err := r.Get(context.TODO(), request.NamespacedName, cluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Flushing nodes", "objectKey", request.NamespacedName)
			k8shandler.FlushNodes(request.NamespacedName.Name, request.NamespacedName.Namespace)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if cluster.Spec.ManagementState == loggingv1.ManagementStateUnmanaged {
		return ctrl.Result{}, nil
	}

	//skip reconciliation if operator is deployed globally but the resource
	//wants to be managed by a namespaced operator
	if isNamespaceManaged(cluster) && !isNamespaceDeployed() {
		return ctrl.Result{}, nil
	}

	if cluster.Spec.Spec.Image != "" {
		if cluster.Status.Conditions == nil {
			cluster.Status.Conditions = []loggingv1.ClusterCondition{}
		}
		exists := false
		for _, condition := range cluster.Status.Conditions {
			if condition.Type == loggingv1.CustomImage {
				exists = true
				break
			}
		}
		if !exists {
			cluster.Status.Conditions = append(cluster.Status.Conditions, loggingv1.ClusterCondition{
				Type:               loggingv1.CustomImage,
				Status:             v1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "CustomImageUnsupported",
				Message:            "Specifiying a custom image from the custom resource is not supported",
			})
		}

	}

	if err = k8shandler.Reconcile(cluster, r.Client); err != nil {
		return reconcileResult, err
	}

	return reconcileResult, nil
}

func (r *ElasticsearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("elasticsearch-controller").
		For(&loggingv1.Elasticsearch{}).
		Complete(r)
}

//isNamespaceManaged determines if a deployment of this operator in a non-global namespace
//should coordinate the resource in lieu of the globally deployed operator
func isNamespaceManaged(cluster *loggingv1.Elasticsearch) bool {
	operatorNS := os.Getenv("POD_NAMESPACE")
	if operatorNS == "" {
		return false
	}
	namespacemanaged := "elasticsearch.openshift.io/namespacemanaged"
	value, found := cluster.Annotations[namespacemanaged]
	if !found || value == "" || "false" == strings.ToLower(value) {
		return false
	}
	return operatorNS == cluster.Namespace
}

func isNamespaceDeployed() bool {
	watchNS := os.Getenv("WATCH_NAMESPACE")
	return watchNS != ""
}
