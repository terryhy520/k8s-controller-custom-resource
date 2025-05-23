package main

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller implements a Kubernetes controller
type Controller struct {
	clientset kubernetes.Interface
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
}

// NewController creates a new controller
func NewController(clientset kubernetes.Interface, informer cache.SharedIndexInformer) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller := &Controller{
		clientset: clientset,
		queue:    queue,
		informer: informer,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	return controller
}

// Run starts the controller
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer c.queue.ShutDown()

	klog.Info("Starting controller")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		klog.Error("Timed out waiting for caches to sync")
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	// Process the key
	err := c.syncHandler(key.(string))
	if err != nil {
		c.queue.AddRateLimited(key)
		klog.Errorf("Error syncing %s: %v", key, err)
	} else {
		c.queue.Forget(key)
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		// Handle deletion
		return nil
	}

	// Handle object
	_ = obj.(runtime.Object)

	return nil
}