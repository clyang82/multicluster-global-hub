package controllers

import (
	"context"
	"fmt"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller interface {
	Reconcile(ctx context.Context, obj interface{}) error
	Run(ctx context.Context)
}

type GenericController struct {
	context context.Context
	name    string
	// client is used to apply resources
	client dynamic.Interface
	// informer is used to watch the hosted resources
	informer cache.Informer

	gvr schema.GroupVersionResource

	queue workqueue.RateLimitingInterface
	cache cache.Cache

	createInstance func() client.Object

	Controller
}

func NewGenericController(ctx context.Context, name string, client dynamic.Interface,
	gvr schema.GroupVersionResource, informer cache.Informer, cache cache.Cache, createInstance func() client.Object) *GenericController {

	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name)
	c := &GenericController{
		context:        ctx,
		name:           name,
		client:         client,
		gvr:            gvr,
		informer:       informer,
		queue:          queue,
		cache:          cache,
		createInstance: createInstance,
	}

	c.informer.AddEventHandler(
		toolscache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.enqueue(obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				c.enqueue(obj)
			},
			DeleteFunc: func(obj interface{}) {
				c.enqueue(obj)
			},
		},
	)
	return c
}

// enqueue enqueues a resource.
func (c *GenericController) enqueue(obj interface{}) {
	key, err := toolscache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

func (c *GenericController) Run(ctx context.Context, numThreads int) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting %s controller", c.name)
	defer klog.Infof("Shutting down %s controller", c.name)

	for i := 0; i < numThreads; i++ {
		go wait.UntilWithContext(ctx, c.startWorker, time.Second)
	}

	<-ctx.Done()
}

func (c *GenericController) startWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *GenericController) processNextWorkItem(ctx context.Context) bool {
	// Wait until there is a new item in the working queue
	k, quit := c.queue.Get()
	if quit {
		return false
	}
	key := k.(string)

	// No matter what, tell the queue we're done with this key, to unblock
	// other workers.
	defer c.queue.Done(key)

	if err := c.process(ctx, key); err != nil {
		utilruntime.HandleError(fmt.Errorf("%q controller failed to sync %q, err: %w", c.name, key, err))
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *GenericController) process(ctx context.Context, key string) error {

	namespace, name, err := toolscache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("invalid key: %q: %v", key, err)
		return nil
	}

	klog.V(5).Infof("process object is: %s/%s", namespace, name)
	instance := c.createInstance()
	err = c.cache.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, instance)
	if err != nil {
		return err
	}

	klog.V(5).Infof("process object is: %v", instance)
	err = c.Reconcile(ctx, instance)
	if err != nil {
		return err
	}

	return nil
}

func (c *GenericController) Reconcile(ctx context.Context, obj interface{}) error {
	klog.Info("Starting to reconcile the resource")

	tempObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	unstructuredObj := &unstructured.Unstructured{Object: tempObj}

	//clean up unneeded fields
	manipulateObj(unstructuredObj)

	if unstructuredObj.GetNamespace() != "" {
		runtimeObj, err := c.client.Resource(c.gvr).Namespace(unstructuredObj.GetNamespace()).
			Get(ctx, unstructuredObj.GetName(), metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				_, err = c.client.Resource(c.gvr).Namespace(unstructuredObj.GetNamespace()).
					Create(ctx, unstructuredObj, metav1.CreateOptions{})
				if err != nil {
					klog.Errorf("failed to create %s: %v", unstructuredObj.GetKind(), err)
					return err
				}
				return nil
			}
			klog.Errorf("failed to list %s: %v", unstructuredObj.GetKind(), err)
			return err
		}
		unstructuredObj.SetResourceVersion(runtimeObj.GetResourceVersion())
		_, err = c.client.Resource(c.gvr).Namespace(unstructuredObj.GetNamespace()).
			Update(ctx, unstructuredObj, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update %s: %v", unstructuredObj.GetKind(), err)
			return err
		}
	} else {
		runtimeObj, err := c.client.Resource(c.gvr).
			Get(ctx, unstructuredObj.GetName(), metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				_, err = c.client.Resource(c.gvr).
					Create(ctx, unstructuredObj, metav1.CreateOptions{})
				if err != nil {
					klog.Errorf("failed to create %s: %v", unstructuredObj.GetKind(), err)
					return err
				}
				return nil
			}
			klog.Errorf("failed to list %s: %v", unstructuredObj.GetKind(), err)
			return err
		}

		unstructuredObj.SetResourceVersion(runtimeObj.GetResourceVersion())
		_, err = c.client.Resource(c.gvr).
			Update(ctx, unstructuredObj, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("failed to update %s: %v", unstructuredObj.GetKind(), err)
			return err
		}
	}

	return nil
}

func manipulateObj(unstructuredObj *unstructured.Unstructured) {
	unstructuredObj.SetUID("")
	unstructuredObj.SetResourceVersion("")
	unstructuredObj.SetManagedFields(nil)
	unstructuredObj.SetFinalizers(nil)
	unstructuredObj.SetGeneration(0)
	unstructuredObj.SetOwnerReferences(nil)

	delete(unstructuredObj.GetAnnotations(), "kubectl.kubernetes.io/last-applied-configuration")

}
