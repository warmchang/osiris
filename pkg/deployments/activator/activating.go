package activator

import (
	"context"
	"encoding/json"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s_types "k8s.io/apimachinery/pkg/types"

	"github.com/dailymotion/osiris/pkg/kubernetes"
)

func (a *activator) activateDeployment(
	ctx context.Context,
	app *app,
) (*appActivation, error) {
	deploymentsClient := a.kubeClient.AppsV1().Deployments(app.namespace)
	deployment, err := deploymentsClient.Get(
		ctx,
		app.name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}
	da := &appActivation{
		readyAppPodIPs: map[string]struct{}{},
		successCh:      make(chan struct{}),
		timeoutCh:      make(chan struct{}),
	}
	glog.Infof(
		"Activating deployment %s in namespace %s",
		app.name,
		app.namespace,
	)
	go da.watchForCompletion(
		a.kubeClient,
		app,
		labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector(),
	)
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas > 0 {
		// We don't need to do this, as it turns out! Scaling is either already
		// in progress-- perhaps initiated by another process-- or may even be
		// completed already. Just return dr and allow the caller to move on to
		// verifying / waiting for this activation to be complete.
		return da, nil
	}
	patches := []kubernetes.PatchOperation{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: kubernetes.GetMinReplicas(deployment.Annotations, 1),
	}}
	patchesBytes, _ := json.Marshal(patches)
	_, err = deploymentsClient.Patch(
		ctx,
		app.name,
		k8s_types.JSONPatchType,
		patchesBytes,
		metav1.PatchOptions{},
	)
	return da, err
}

func (a *activator) activateStatefulSet(
	ctx context.Context,
	app *app,
) (*appActivation, error) {
	statefulSetsClient := a.kubeClient.AppsV1().StatefulSets(app.namespace)
	statefulSet, err := statefulSetsClient.Get(
		ctx,
		app.name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}
	da := &appActivation{
		readyAppPodIPs: map[string]struct{}{},
		successCh:      make(chan struct{}),
		timeoutCh:      make(chan struct{}),
	}
	glog.Infof(
		"Activating statefulSet %s in namespace %s",
		app.name,
		app.namespace,
	)
	go da.watchForCompletion(
		a.kubeClient,
		app,
		labels.Set(statefulSet.Spec.Selector.MatchLabels).AsSelector(),
	)
	if statefulSet.Spec.Replicas == nil || *statefulSet.Spec.Replicas > 0 {
		// We don't need to do this, as it turns out! Scaling is either already
		// in progress-- perhaps initiated by another process-- or may even be
		// completed already. Just return dr and allow the caller to move on to
		// verifying / waiting for this activation to be complete.
		return da, nil
	}
	patches := []kubernetes.PatchOperation{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: kubernetes.GetMinReplicas(statefulSet.Annotations, 1),
	}}
	patchesBytes, _ := json.Marshal(patches)
	_, err = statefulSetsClient.Patch(
		ctx,
		app.name,
		k8s_types.JSONPatchType,
		patchesBytes,
		metav1.PatchOptions{},
	)
	return da, err
}
