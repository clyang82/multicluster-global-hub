/*
Copyright 2021 The KCP Authors.

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

package syncer

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

func deepEqualStatus(oldObj, newObj interface{}) bool {
	oldUnstrob, isOldObjUnstructured := oldObj.(*unstructured.Unstructured)
	newUnstrob, isNewObjUnstructured := newObj.(*unstructured.Unstructured)
	if !isOldObjUnstructured || !isNewObjUnstructured || oldObj == nil || newObj == nil {
		return false
	}

	newStatus := newUnstrob.UnstructuredContent()["status"]
	oldStatus := oldUnstrob.UnstructuredContent()["status"]

	return equality.Semantic.DeepEqual(oldStatus, newStatus)
}

const statusSyncerAgent = "hoh#status-syncer/v0.0.0"

func NewStatusSyncer(from, to *rest.Config) (*Controller, error) {
	from = rest.CopyConfig(from)
	from.UserAgent = statusSyncerAgent
	to = rest.CopyConfig(to)
	to.UserAgent = statusSyncerAgent

	fromClient := dynamic.NewForConfigOrDie(from)
	toClient := dynamic.NewForConfigOrDie(to)

	return New(fromClient, toClient, SyncUp)
}

func (c *Controller) updateStatusInUpstream(ctx context.Context, gvr schema.GroupVersionResource, upstreamNamespace string, downstreamObj *unstructured.Unstructured) error {
	upstreamObj := downstreamObj.DeepCopy()
	upstreamObj.SetUID("")
	upstreamObj.SetResourceVersion("")
	upstreamObj.SetNamespace(upstreamNamespace)

	existing, err := c.toClient.Resource(gvr).Namespace(upstreamNamespace).Get(ctx, upstreamObj.GetName(), metav1.GetOptions{})
	if err != nil {
		if gvr.Resource == "managedclusters" && errors.IsNotFound(err) {
			c.applyToUpstream(ctx, gvr, upstreamNamespace, downstreamObj)
		}
		klog.Errorf("Getting resource %s/%s: %v", upstreamNamespace, upstreamObj.GetName(), err)
		return err
	}

	upstreamObj.SetResourceVersion(existing.GetResourceVersion())
	if _, err := c.toClient.Resource(gvr).Namespace(upstreamNamespace).UpdateStatus(ctx, upstreamObj, metav1.UpdateOptions{}); err != nil {
		klog.Errorf("Failed updating status of resource %s/%s from leaf hub cluster namespace %s: %v", upstreamNamespace, upstreamObj.GetName(), downstreamObj.GetNamespace(), err)
		return err
	}
	klog.Infof("Updated status of resource %s/%s from leaf hub cluster namespace %s", upstreamNamespace, upstreamObj.GetName(), downstreamObj.GetNamespace())

	return nil
}

// applyToUpstream is used to apply managedclusters to upstream
func (c *Controller) applyToUpstream(ctx context.Context, gvr schema.GroupVersionResource, upstreamNamespace string, downstreamObj *unstructured.Unstructured) error {

	upstreamObj := downstreamObj.DeepCopy()
	upstreamObj.SetUID("")
	upstreamObj.SetResourceVersion("")
	upstreamObj.SetManagedFields(nil)
	// Deletion fields are immutable and set by the downstream API server
	upstreamObj.SetDeletionTimestamp(nil)
	upstreamObj.SetDeletionGracePeriodSeconds(nil)
	// Strip owner references, to avoid orphaning by broken references,
	// and make sure cascading deletion is only performed once upstream.
	upstreamObj.SetOwnerReferences(nil)
	// Strip finalizers to avoid the deletion of the downstream resource from being blocked.
	upstreamObj.SetFinalizers(nil)

	// Marshalling the unstructured object is good enough as SSA patch
	data, err := json.Marshal(upstreamObj)
	if err != nil {
		return err
	}

	if _, err := c.toClient.Resource(gvr).Patch(ctx, upstreamObj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: syncerApplyManager, Force: pointer.Bool(true)}); err != nil {
		klog.Infof("Error upserting %s %s from downstream %s: %v", gvr.Resource, upstreamObj.GetName(), downstreamObj.GetName(), err)
		return err
	}
	klog.Infof("Upserted %s %s from upstream %s", gvr.Resource, upstreamObj.GetName(), downstreamObj.GetName())

	return nil
}
