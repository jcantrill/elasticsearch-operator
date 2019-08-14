package k8shandler

import (
	"testing"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateResourcesWhenDesiredCPULimitIsZero(t *testing.T) {
	nodeContainer := v1.Container{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourceCPU:    resource.MustParse("600m"),
			},
			Requests: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourceCPU:    resource.MustParse("600m"),
			},
		},
	}
	desiredContainer := v1.Container{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Requests: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourceCPU:    resource.MustParse("600m"),
			},
		},
	}
	deployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: apps.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						nodeContainer,
					},
				},
			},
		},
	}
	node := &deploymentNode{
		self: deployment,
	}
	actual, changed := updateResources(node, nodeContainer, desiredContainer)

	if !changed {
		t.Error("Expected updating the resources would recognized as changed, but it was not")
	}
	if !areResourcesSame(actual.Resources, desiredContainer.Resources) {
		t.Errorf("Expected %v but got %v", printResource(desiredContainer.Resources), printResource(actual.Resources))
	}
}
func TestUpdateResourcesWhenDesiredMemoryLimitIsZero(t *testing.T) {
	nodeContainer := v1.Container{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourceCPU:    resource.MustParse("600m"),
			},
			Requests: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourceCPU:    resource.MustParse("600m"),
			},
		},
	}
	desiredContainer := v1.Container{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("600m"),
			},
			Requests: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourceCPU:    resource.MustParse("600m"),
			},
		},
	}
	deployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: apps.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						nodeContainer,
					},
				},
			},
		},
	}
	node := &deploymentNode{
		self: deployment,
	}
	actual, changed := updateResources(node, nodeContainer, desiredContainer)

	if !changed {
		t.Error("Expected updating the resources would recognized as changed, but it was not")
	}
	if !areResourcesSame(actual.Resources, desiredContainer.Resources) {
		t.Errorf("Expected %v but got %v", printResource(desiredContainer.Resources), printResource(actual.Resources))
	}
}
