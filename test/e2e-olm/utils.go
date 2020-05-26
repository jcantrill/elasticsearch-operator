package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	consolev1 "github.com/openshift/api/console/v1"
	elasticsearch "github.com/openshift/elasticsearch-operator/pkg/apis/logging/v1"
	"github.com/openshift/elasticsearch-operator/pkg/constants"
	"github.com/openshift/elasticsearch-operator/test/utils"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

const (
	retryInterval        = time.Second * 1
	timeout              = time.Second * 300
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
	elasticsearchCRName  = "elasticsearch"
	kibanaCRName         = "kibana"
)

func registerSchemes(t *testing.T) {
	elasticsearchList := &elasticsearch.ElasticsearchList{}
	err := framework.AddToFrameworkScheme(elasticsearch.SchemeBuilder.AddToScheme, elasticsearchList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	kibanaList := &elasticsearch.KibanaList{}
	err = framework.AddToFrameworkScheme(elasticsearch.SchemeBuilder.AddToScheme, kibanaList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	consoleLinkList := &consolev1.ConsoleLinkList{}
	err = framework.AddToFrameworkScheme(consolev1.Install, consoleLinkList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
}

func createElasticsearchCR(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, uuid string) (*elasticsearch.Elasticsearch, error) {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return nil, fmt.Errorf("Could not get namespace: %v", err)
	}

	cpuValue := resource.MustParse("100m")
	memValue := resource.MustParse("2Gi")

	esDataNode := elasticsearch.ElasticsearchNode{
		Roles: []elasticsearch.ElasticsearchNodeRole{
			elasticsearch.ElasticsearchRoleClient,
			elasticsearch.ElasticsearchRoleData,
			elasticsearch.ElasticsearchRoleMaster,
		},
		NodeCount: int32(1),
		Storage:   elasticsearch.ElasticsearchStorageSpec{},
		GenUUID:   &uuid,
	}

	// create elasticsearch custom resource
	cr := &elasticsearch.Elasticsearch{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Elasticsearch",
			APIVersion: elasticsearch.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      elasticsearchCRName,
			Namespace: namespace,
			Annotations: map[string]string{
				"elasticsearch.openshift.io/develLogAppender": "console",
				"elasticsearch.openshift.io/loglevel":         "trace",
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

	t.Logf("Creating Elasticsearch CR: %v", cr)
	err = f.Client.Create(goctx.TODO(), cr, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return nil, fmt.Errorf("could not create exampleElasticsearch: %v", err)
	}

	return cr, nil
}

// Create the secret that would be generated by CLO normally
func createElasticsearchSecret(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	t.Log("Creating required secret")
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

	t.Logf("Creating %v", elasticsearchSecret)
	err = f.Client.Create(goctx.TODO(), elasticsearchSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	return nil
}

func updateElasticsearchSecret(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
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

	t.Log("Updating required secret...")
	err = f.Client.Update(goctx.TODO(), elasticsearchSecret)
	if err != nil {
		return err
	}

	return nil
}

func createKibanaCR(namespace string) *elasticsearch.Kibana {
	cpuValue, _ := resource.ParseQuantity("100m")
	memValue, _ := resource.ParseQuantity("256Mi")

	return &elasticsearch.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.KibanaInstanceName,
			Namespace: namespace,
		},
		Spec: elasticsearch.KibanaSpec{
			ManagementState: elasticsearch.ManagementStateManaged,
			Replicas:        1,
			Resources: &v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceMemory: memValue,
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    cpuValue,
					v1.ResourceMemory: memValue,
				},
			},
		},
	}
}

func createKibanaSecret(f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)

	}

	kibanaSecret := utils.Secret(
		kibanaCRName,
		namespace,
		map[string][]byte{
			"key":  utils.GetFileContents("/tmp/example-secrets/system.logging.kibana.key"),
			"cert": utils.GetFileContents("/tmp/example-secrets/system.logging.kibana.crt"),
			"ca":   utils.GetFileContents("/tmp/example-secrets/ca.crt"),
		},
	)

	err = f.Client.Create(goctx.TODO(), kibanaSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	return nil
}

func createKibanaProxySecret(f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)

	}

	kibanaProxySecret := utils.Secret(
		fmt.Sprintf("%s-proxy", kibanaCRName),
		namespace,
		map[string][]byte{
			"server-key":     utils.GetFileContents("/tmp/example-secrets/kibana-internal.key"),
			"server-cert":    utils.GetFileContents("/tmp/example-secrets/kibana-internal.crt"),
			"session-secret": []byte("TG85VUMyUHBqbWJ1eXo1R1FBOGZtYTNLTmZFWDBmbkY="),
		},
	)

	err = f.Client.Create(goctx.TODO(), kibanaProxySecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	return nil
}