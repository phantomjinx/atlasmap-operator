// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package e2e

import (
	"context"
	goctx "context"
	"fmt"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/atlasmap/atlasmap-operator/pkg/apis"
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/controller/atlasmap"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 30
)

func TestAtlasMap(t *testing.T) {
	atlasMapList := &v1alpha1.AtlasMapList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasMap",
			APIVersion: "atlasmap.io/v1alpha1",
		},
	}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, atlasMapList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	// run subtests
	t.Run("atlasmap-group", func(t *testing.T) {
		t.Run("Cluster", AtlasMapCluster)
	})
}

func atlasMapDeploymentTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-deployment"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasMap",
			APIVersion: "atlasmap.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas: 1,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	err = f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap)
	if err != nil {
		return err
	}

	if exampleAtlasMap.Status.Image != atlasmap.DefaultImageName {
		return fmt.Errorf("Expected AtlasMap.Status.Image to be %s but was %s", atlasmap.DefaultImageName, exampleAtlasMap.Status.Image)
	}

	// Verify a service was created
	atlasMapService := &v1.Service{}
	err = f.Client.Get(context.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, atlasMapService)
	if err != nil {
		return err
	}

	// Verify a route was created
	atlasMapRoute := &routev1.Route{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, atlasMapRoute)
	if err != nil {
		return err
	}

	expectedURL := "https://" + atlasMapRoute.Spec.Host
	if exampleAtlasMap.Status.URL != expectedURL {
		return fmt.Errorf("Expected AtlasMap.Status.URL to be %s but was %s", expectedURL, exampleAtlasMap.Status.URL)
	}

	return nil
}

func atlasMapScaleTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-scale"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasMap",
			APIVersion: "atlasmap.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas: 1,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	err = f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap)
	if err != nil {
		return err
	}

	exampleAtlasMap.Spec.Replicas = 3
	err = f.Client.Update(goctx.TODO(), exampleAtlasMap)
	if err != nil {
		return err
	}

	// wait for deployment to reach 3 replicas
	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 3, retryInterval, timeout)
}

func atlasMapImageNameTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-custom-image"
	imageName := "docker.io/jamesnetherton/atlasmap:latest"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasMap",
			APIVersion: "atlasmap.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas: 1,
			Image:    imageName,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	err = f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap)
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	err = f.Client.Get(context.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, deployment)
	if err != nil {
		return err
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != exampleAtlasMap.Spec.Image {
		return fmt.Errorf("Expected container image to match %s but got %s", exampleAtlasMap.Spec.Image, container.Image)
	}

	return nil
}

func atlasMapResourcesTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()

	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-resources"
	limitCPU := "700m"
	limitMemory := "512Mi"
	requestCPU := "500m"
	requestMemory := "256Mi"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasMap",
			APIVersion: "atlasmap.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas:      1,
			LimitCPU:      limitCPU,
			LimitMemory:   limitMemory,
			RequestCPU:    requestCPU,
			RequestMemory: requestMemory,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	err = f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout*2)
	if err != nil {
		return err
	}

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap)
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	err = f.Client.Get(context.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, deployment)
	if err != nil {
		return err
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	if container.Resources.Limits.Cpu().String() != limitCPU {
		return fmt.Errorf("Expected CPU limit to match %s but got %s", limitCPU, container.Resources.Limits.Cpu().String())
	}

	if container.Resources.Limits.Memory().String() != limitMemory {
		return fmt.Errorf("Expected memory limit to match %s but got %s", limitMemory, container.Resources.Limits.Memory().String())
	}

	if container.Resources.Requests.Cpu().String() != requestCPU {
		return fmt.Errorf("Expected CPU request to match %s but got %s", requestCPU, container.Resources.Requests.Cpu().String())
	}

	if container.Resources.Requests.Memory().String() != requestMemory {
		return fmt.Errorf("Expected memory request to match %s but got %s", requestMemory, container.Resources.Requests.Memory().String())
	}

	return nil
}

func AtlasMapCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}

	t.Log("Initialized cluster resources")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	f := framework.Global
	routev1.AddToScheme(framework.Global.Scheme)

	if !f.LocalOperator {
		err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "atlasmap-operator", 1, retryInterval, timeout)
		if err != nil {
			t.Fatal(err)
		}
	}

	type testFunction func(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error

	tests := []testFunction{
		atlasMapDeploymentTest,
		atlasMapScaleTest,
		atlasMapImageNameTest,
		atlasMapResourcesTest,
	}

	// run tests
	for _, test := range tests {
		if err = test(t, f, ctx); err != nil {
			t.Fatal(err)
		}
	}
}
