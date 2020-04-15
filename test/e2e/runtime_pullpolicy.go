package e2e

import (
	goctx "context"
	"errors"
	"testing"
	"time"

	appstacksv1beta1 "github.com/application-stacks/runtime-component-operator/pkg/apis/appstacks/v1beta1"
	"github.com/application-stacks/runtime-component-operator/test/util"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	e2eutil "github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	k "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// RuntimePullPolicyTest checks that the configured pull policy is applied to deployment
func RuntimePullPolicyTest(t *testing.T) {
	ctx, err := util.InitializeContext(t, cleanupTimeout, retryInterval)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Cleanup()

	f := framework.Global
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %v", err)
	}
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "runtime-component-operator", 1, retryInterval, operatorTimeout)
	if err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}
	timestamp := time.Now().UTC()
	t.Logf("%s - Starting runtime pull policy test...", timestamp)

	// create one replica of the operator deployment in current namespace with provided name
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "runtime-component-operator", 1, retryInterval, operatorTimeout)
	if err != nil {
		t.Fatal(err)
	}

	replicas := int32(1)
	policy := k.PullAlways

	runtimeComponent := util.MakeBasicRuntimeComponent(t, f, "example-runtime-pullpolicy", namespace, replicas)
	runtimeComponent.Spec.PullPolicy = &policy

	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), runtimeComponent, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}

	// wait for example-runtime-pullpolicy to reach 2 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-runtime-pullpolicy", 1, retryInterval, timeout)
	if err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}

	timestamp = time.Now().UTC()
	t.Logf("%s - Deployment created, verifying pull policy...", timestamp)

	if err = verifyPullPolicy(t, f, "example-runtime-pullpolicy", namespace, "Always"); err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}

	if err = updatePullPolicy(t, f, ctx); err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}

	if err = deletePullPolicy(t, f, ctx); err != nil {
		util.FailureCleanup(t, f, namespace, err)
	}
}

func verifyPullPolicy(t *testing.T, f *framework.Framework, name, namespace string, policy k.PullPolicy) error {

	deploy, err := f.KubeClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Logf("Got error when getting PullPolicy %s: %s", name, err)
		return err
	}

	if deploy.Spec.Template.Spec.Containers[0].ImagePullPolicy != policy {
		return errors.New("pull policy was not successfully configured from the default value")
	}
	return nil
}

func updatePullPolicy(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	const name = "example-runtime-pullpolicy"

	ns, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	target := types.NamespacedName{Namespace: ns, Name: name}
	err = util.UpdateApplication(f, target, func(r *appstacksv1beta1.RuntimeComponent) {
		policy := k.PullNever
		r.Spec.PullPolicy = &policy
	})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, ns, name, 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = verifyPullPolicy(t, f, name, ns, "Never")
	if err != nil {
		return err
	}

	return nil
}

func deletePullPolicy(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	const name = "example-runtime-pullpolicy"

	ns, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	target := types.NamespacedName{Namespace: ns, Name: name}
	err = util.UpdateApplication(f, target, func(r *appstacksv1beta1.RuntimeComponent) {
		r.Spec.PullPolicy = nil
	})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, ns, name, 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = verifyPullPolicy(t, f, name, ns, "IfNotPresent")
	if err != nil {
		return err
	}

	return nil
}
