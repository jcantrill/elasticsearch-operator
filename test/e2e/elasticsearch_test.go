package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/openshift/elasticsearch-operator/test/utils"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	goctx "context"
	elasticsearch "github.com/openshift/elasticsearch-operator/pkg/apis/logging/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	retryInterval        = time.Second * 2
	timeout              = time.Second * 300
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
	elasticsearchCRName  = "elasticsearch"
)

func TestElasticsearch(t *testing.T) {
	elasticsearchList := &elasticsearch.ElasticsearchList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Elasticsearch",
			APIVersion: elasticsearch.SchemeGroupVersion.String(),
		},
	}
	err := framework.AddToFrameworkScheme(elasticsearch.SchemeBuilder.AddToScheme, elasticsearchList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("elasticsearch-group", func(t *testing.T) {
		t.Run("Cluster", ElasticsearchCluster)
	})
}

// Create the secret that would be generated by CLO normally
func createRequiredSecret(f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)
	}

	elasticsearchSecret := utils.Secret(
		elasticsearchCRName,
		namespace,
		map[string][]byte{
			"elasticsearch.key": utils.GetFileContents("/tmp/example-secrets/elasticsearch.key"),
			"elasticsearch.crt": utils.GetFileContents("/tmp/example-secrets/elasticsearch.crt"),
			"logging-es.key":    utils.GetFileContents("/tmp/example-secrets/logging-es.key"),
			"logging-es.crt":    utils.GetFileContents("/tmp/example-secrets/logging-es.crt"),
			"admin-key":         utils.GetFileContents("/tmp/example-secrets/system.admin.key"),
			"admin-cert":        utils.GetFileContents("/tmp/example-secrets/system.admin.crt"),
			"admin-ca":          utils.GetFileContents("/tmp/example-secrets/ca.crt"),
		},
	)

	err = f.Client.Create(goctx.TODO(), elasticsearchSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	return nil
}

func updateRequiredSecret(f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)
	}

	elasticsearchSecret := &v1.Secret{}

	secretName := types.NamespacedName{Name: elasticsearchCRName, Namespace: namespace}
	if err = f.Client.Get(goctx.TODO(), secretName, elasticsearchSecret); err != nil {
		return fmt.Errorf("Could not get secret %s: %v", elasticsearchCRName, err)
	}

	elasticsearchSecret.Data = map[string][]byte{
		"elasticsearch.key": utils.GetFileContents("/tmp/example-secrets/elasticsearch.key"),
		"elasticsearch.crt": utils.GetFileContents("/tmp/example-secrets/elasticsearch.crt"),
		"logging-es.key":    utils.GetFileContents("/tmp/example-secrets/logging-es.key"),
		"logging-es.crt":    utils.GetFileContents("/tmp/example-secrets/logging-es.crt"),
		"admin-key":         utils.GetFileContents("/tmp/example-secrets/system.admin.key"),
		"admin-cert":        utils.GetFileContents("/tmp/example-secrets/system.admin.crt"),
		"admin-ca":          utils.GetFileContents("/tmp/example-secrets/ca.crt"),
		"dummy":             []byte("blah"),
	}

	err = f.Client.Update(goctx.TODO(), elasticsearchSecret)
	if err != nil {
		return err
	}

	return nil
}

func elasticsearchFullClusterTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)
	}

	cpuValue, _ := resource.ParseQuantity("100m")
	memValue, _ := resource.ParseQuantity("2Gi")

	dataUUID := utils.GenerateUUID()

	esDataNode := elasticsearch.ElasticsearchNode{
		Roles: []elasticsearch.ElasticsearchNodeRole{
			elasticsearch.ElasticsearchRoleClient,
			elasticsearch.ElasticsearchRoleData,
			elasticsearch.ElasticsearchRoleMaster,
		},
		NodeCount: int32(1),
		Storage:   elasticsearch.ElasticsearchStorageSpec{},
		GenUUID:   &dataUUID,
	}

	nonDataUUID := utils.GenerateUUID()

	esNonDataNode := elasticsearch.ElasticsearchNode{
		Roles: []elasticsearch.ElasticsearchNodeRole{
			elasticsearch.ElasticsearchRoleClient,
			elasticsearch.ElasticsearchRoleMaster,
		},
		NodeCount: int32(1),
		Storage:   elasticsearch.ElasticsearchStorageSpec{},
		GenUUID:   &nonDataUUID,
	}

	// create clusterlogging custom resource
	exampleElasticsearch := &elasticsearch.Elasticsearch{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Elasticsearch",
			APIVersion: elasticsearch.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      elasticsearchCRName,
			Namespace: namespace,
			Annotations: map[string]string{
				"elasticsearch.openshift.io/develLogAppender": "console",
			},
		},
		Spec: elasticsearch.ElasticsearchSpec{
			Spec: elasticsearch.ElasticsearchNodeSpec{
				Image: "",
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceMemory: memValue,
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:    cpuValue,
						v1.ResourceMemory: memValue,
					},
				},
			},
			Nodes: []elasticsearch.ElasticsearchNode{
				esDataNode,
			},
			ManagementState:  elasticsearch.ManagementStateManaged,
			RedundancyPolicy: elasticsearch.ZeroRedundancy,
		},
	}
	err = f.Client.Create(goctx.TODO(), exampleElasticsearch, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return fmt.Errorf("could not create exampleElasticsearch: %v", err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), err)
	}
	t.Log("Created initial deployment")

	// Scale up current node
	// then look for elasticsearch-cdm-0-2 and prior node
	exampleName := types.NamespacedName{Name: elasticsearchCRName, Namespace: namespace}
	if err = f.Client.Get(goctx.TODO(), exampleName, exampleElasticsearch); err != nil {
		return fmt.Errorf("failed to get exampleElasticsearch: %v", err)
	}
	exampleElasticsearch.Spec.Nodes[0].NodeCount = int32(2)
	err = f.Client.Update(goctx.TODO(), exampleElasticsearch)
	if err != nil {
		return fmt.Errorf("could not update exampleElasticsearch with 2 replicas: %v", err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-2", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-2", dataUUID), err)
	}
	t.Log("Created additional deployment")

	if err = f.Client.Get(goctx.TODO(), exampleName, exampleElasticsearch); err != nil {
		return fmt.Errorf("failed to get exampleElasticsearch: %v", err)
	}
	exampleElasticsearch.Spec.Nodes = append(exampleElasticsearch.Spec.Nodes, esNonDataNode)
	err = f.Client.Update(goctx.TODO(), exampleElasticsearch)
	if err != nil {
		return fmt.Errorf("could not update exampleElasticsearch with an additional node: %v", err)
	}

	// Create another node
	// then look for elasticsearch-cdm-1-1 and prior nodes
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-2", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), err)
	}

	err = utils.WaitForStatefulset(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cm-%v", nonDataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Statefulset %v: %v", fmt.Sprintf("elasticsearch-cm-%v", nonDataUUID), err)
	}
	t.Log("Created non-data statefulset")

	// Scale up to SingleRedundancy
	if err = f.Client.Get(goctx.TODO(), exampleName, exampleElasticsearch); err != nil {
		return fmt.Errorf("failed to get exampleElasticsearch: %v", err)
	}
	exampleElasticsearch.Spec.RedundancyPolicy = elasticsearch.SingleRedundancy
	err = f.Client.Update(goctx.TODO(), exampleElasticsearch)
	if err != nil {
		return fmt.Errorf("could not update exampleElasticsearch to be SingleRedundancy: %v", err)
	}

	/*
		FIXME: this is commented out as we currently do not run our e2e tests in a container on the test cluster
		 to be added back in as a follow up
		err = utils.WaitForIndexTemplateReplicas(t, f.KubeClient, namespace, "elasticsearch", 1, retryInterval, timeout)
		if err != nil {
			return fmt.Errorf("timed out waiting for all index templates to have correct replica count")
		}

		err = utils.WaitForIndexReplicas(t, f.KubeClient, namespace, "elasticsearch", 1, retryInterval, timeout)
		if err != nil {
			return fmt.Errorf("timed out waiting for all indices to have correct replica count")
		}
	*/

	// Update the secret to force a full cluster redeploy
	err = updateRequiredSecret(f, ctx)
	if err != nil {
		return fmt.Errorf("Unable to update secret")
	}

	// wait for pods to have "redeploy for certs" condition as true?
	desiredCondition := elasticsearch.ElasticsearchNodeUpgradeStatus{
		ScheduledForCertRedeploy: v1.ConditionTrue,
	}

	utils.WaitForNodeStatusCondition(t, f, namespace, elasticsearchCRName, desiredCondition, retryInterval, time.Second*30)
	if err != nil {
		return fmt.Errorf("Timed out waiting for full cluster restart to begin")
	}

	// then wait for conditions to be gone
	desiredClusterCondition := elasticsearch.ClusterCondition{
		Type:   elasticsearch.Restarting,
		Status: v1.ConditionFalse,
	}
	utils.WaitForClusterStatusCondition(t, f, namespace, elasticsearchCRName, desiredClusterCondition, retryInterval, time.Second*300)
	if err != nil {
		return fmt.Errorf("Timed out waiting for full cluster restart to complete")
	}

	// ensure all prior nodes are ready again
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), err)
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cdm-%v-2", dataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Deployment %v: %v", fmt.Sprintf("elasticsearch-cdm-%v-1", dataUUID), err)
	}

	err = utils.WaitForStatefulset(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cm-%v", nonDataUUID), 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("timed out waiting for Statefulset %v: %v", fmt.Sprintf("elasticsearch-cm-%v", nonDataUUID), err)
	}

	// Incorrect scale up and verify we don't see a 4th master created
	if err = f.Client.Get(goctx.TODO(), exampleName, exampleElasticsearch); err != nil {
		return fmt.Errorf("failed to get exampleElasticsearch: %v", err)
	}
	exampleElasticsearch.Spec.Nodes[1].NodeCount = int32(2)
	err = f.Client.Update(goctx.TODO(), exampleElasticsearch)
	if err != nil {
		return fmt.Errorf("could not update exampleElasticsearch with an additional statefulset replica: %v", err)
	}

	err = utils.WaitForStatefulset(t, f.KubeClient, namespace, fmt.Sprintf("elasticsearch-cm-%v", nonDataUUID), 2, retryInterval, time.Second*30)
	if err == nil {
		return fmt.Errorf("unexpected statefulset replica count for %v found", fmt.Sprintf("elasticsearch-cm-%v", nonDataUUID))
	}

	if err = f.Client.Get(goctx.TODO(), exampleName, exampleElasticsearch); err != nil {
		return fmt.Errorf("failed to get exampleElasticsearch: %v", err)
	}

	for _, condition := range exampleElasticsearch.Status.Conditions {
		if condition.Type == elasticsearch.InvalidMasters {
			if condition.Status == v1.ConditionFalse ||
				condition.Status == "" {
				return fmt.Errorf("unexpected status condition for elasticsearch found: %v", condition.Status)
			}
		}
	}

	t.Log("Finished successfully")
	return nil
}

func ElasticsearchCluster(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Found namespace: %v", namespace)

	// get global framework variables
	f := framework.Global
	// wait for elasticsearch-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "elasticsearch-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = createRequiredSecret(f, ctx); err != nil {
		t.Fatal(err)
	}

	if err = elasticsearchFullClusterTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
