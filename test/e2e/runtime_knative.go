package e2e

import (
	goctx "context"
	"errors"
	"testing"
	"time"

	appstacksv1beta1 "github.com/application-stacks/runtime-component-operator/pkg/apis/appstacks/v1beta1"
	"github.com/application-stacks/runtime-component-operator/test/util"
	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	e2eutil "github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// RuntimeKnativeTest : Create application with knative service enabled to verify feature
func RuntimeKnativeTest(t *testing.T) {
	ctx, err := util.InitializeContext(t, cleanupTimeout, retryInterval)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Cleanup()
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("Couldn't get namespace: %v", err)
	}

	t.Logf("Namespace: %s", namespace)

	f := framework.Global

	// catch cases where running tests locally with a cluster that does not have knative
	if !isKnativeInstalled(t, f) {
		t.Log("Knative is not installed on this cluster, skipping RuntimeKnativeTest...")
		return
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "runtime-component-operator", 1, retryInterval, operatorTimeout)
	if err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}
	knativeBool := true
	applicationName := "example-runtime-knative"

	exampleRuntime := util.MakeBasicRuntimeComponent(t, f, applicationName, namespace, 1)
	exampleRuntime.Spec.CreateKnativeService = &knativeBool

	// Create application deployment and wait
	err = f.Client.Create(goctx.TODO(), exampleRuntime, &framework.CleanupOptions{TestContext: ctx, Timeout: time.Second, RetryInterval: time.Second})
	if err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}

	err = util.WaitForKnativeDeployment(t, f, namespace, applicationName, retryInterval, timeout)
	if err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}

	if err = disableKnativeTest(t, f, ctx); err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}
}

func disableKnativeTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	ns, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	knativeBool := false
	const name = "example-runtime-knative"
	target := types.NamespacedName{Name: name, Namespace: ns}
	err = util.UpdateApplication(f, target,  func(r *appstacksv1beta1.RuntimeComponent) {
		r.Spec.CreateKnativeService = &knativeBool
	})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, ns, name, 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	ksvc := servingv1alpha1.Service{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: ns, Name: name}, &ksvc)
	if err == nil {
		return errors.New("knative service not deleted")
	}

	return nil
}

func isKnativeInstalled(t *testing.T, f *framework.Framework) bool {
	ns := &corev1.NamespaceList{}
	err := f.Client.List(goctx.TODO(), ns)
	if err != nil {
		t.Fatalf("Error occurred while trying to find knative-serving %v", err)
	}
	for _, val := range ns.Items {
		if val.Name == "knative-serving" {
			return true
		}
	}
	return false
}
