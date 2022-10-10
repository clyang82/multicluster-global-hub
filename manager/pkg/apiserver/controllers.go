package apiserver

import (
	"context"
	"fmt"

	filteredcache "github.com/IBM/controller-filtered-cache/filteredcache"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	policyv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stolostron/multicluster-global-hub/manager/pkg/apiserver/controllers"
	"github.com/stolostron/multicluster-global-hub/pkg/constants"
)

func (s *GlobalHubApiServer) CreateCache(ctx context.Context) error {
	scheme := runtime.NewScheme()

	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return err
	}
	if err := policyv1.AddToScheme(scheme); err != nil {
		return err
	}
	if err := placementrulev1.AddToScheme(scheme); err != nil {
		return err
	}
	if err := clusterv1beta1.AddToScheme(scheme); err != nil {
		return err
	}

	gvkLabelsMap := map[schema.GroupVersionKind][]filteredcache.Selector{
		apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"): {
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "policies.policy.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "placementbindings.policy.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "placementrules.apps.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "managedclusters.cluster.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "subscriptionreports.apps.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "subscriptions.apps.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "subscriptionstatuses.apps.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "placements.cluster.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "managedclustersetbindings.cluster.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "managedclustersets.cluster.open-cluster-management.io")},

			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "clusterdeployments.hive.openshift.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "machinepools.hive.openshift.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "klusterletaddonconfigs.agent.open-cluster-management.io")},
		},
		policyv1.SchemeGroupVersion.WithKind("Policy"): {
			{LabelSelector: fmt.Sprint("!" + constants.GlobalHubLocalResource)},
		},
		policyv1.SchemeGroupVersion.WithKind("PlacementBinding"): {
			{LabelSelector: fmt.Sprint("!" + constants.GlobalHubLocalResource)},
		},
		placementrulev1.SchemeGroupVersion.WithKind("PlacementRule"): {
			{LabelSelector: fmt.Sprint("!" + constants.GlobalHubLocalResource)},
		},
		clusterv1beta1.SchemeGroupVersion.WithKind("Placement"): {
			{LabelSelector: fmt.Sprint("!" + constants.GlobalHubLocalResource)},
		},
	}

	opts := cache.Options{
		Scheme: scheme,
	}

	var err error
	s.Cache, err = filteredcache.NewEnhancedFilteredCacheBuilder(gvkLabelsMap)(s.hostedConfig, opts)
	if err != nil {
		return err
	}

	go s.Cache.Start(ctx)
	s.Cache.WaitForCacheSync(ctx)
	return nil
}

func (s *GlobalHubApiServer) InstallCRDController(ctx context.Context, config *rest.Config) error {
	controllerName := "global-hub-crd-controller"
	config = rest.AddUserAgent(rest.CopyConfig(config), controllerName)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	informer, err := s.Cache.GetInformerForKind(ctx,
		apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"))
	if err != nil {
		return err
	}
	// configure the dynamic informer event handlers
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		apiextensionsv1.SchemeGroupVersion.WithResource("customresourcedefinitions"), informer, s.Cache,
		func() client.Object { return &apiextensionsv1.CustomResourceDefinition{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(
		hookContext genericapiserver.PostStartHookContext,
	) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPolicyController(ctx context.Context, config *rest.Config) error {
	controllerName := "global-hub-policy-controller"
	config = rest.AddUserAgent(rest.CopyConfig(config), controllerName)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	informer, err := s.Cache.GetInformerForKind(ctx,
		policyv1.SchemeGroupVersion.WithKind("Policy"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		policyv1.SchemeGroupVersion.WithResource("policies"), informer, s.Cache,
		func() client.Object { return &policyv1.Policy{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(
		hookContext genericapiserver.PostStartHookContext,
	) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPlacementRuleController(ctx context.Context, config *rest.Config) error {
	controllerName := "global-hub-placementrule-controller"
	config = rest.AddUserAgent(rest.CopyConfig(config), controllerName)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	informer, err := s.Cache.GetInformerForKind(ctx,
		placementrulev1.SchemeGroupVersion.WithKind("PlacementRule"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		placementrulev1.SchemeGroupVersion.WithResource("placementrules"), informer, s.Cache,
		func() client.Object { return &placementrulev1.PlacementRule{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(
		hookContext genericapiserver.PostStartHookContext,
	) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPlacementController(ctx context.Context, config *rest.Config) error {
	controllerName := "global-hub-placements-controller"
	config = rest.AddUserAgent(rest.CopyConfig(config), controllerName)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	informer, err := s.Cache.GetInformerForKind(ctx,
		clusterv1beta1.SchemeGroupVersion.WithKind("Placement"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		clusterv1beta1.SchemeGroupVersion.WithResource("placements"), informer, s.Cache,
		func() client.Object { return &clusterv1beta1.Placement{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(
		hookContext genericapiserver.PostStartHookContext,
	) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPlacementBindingController(ctx context.Context, config *rest.Config) error {
	controllerName := "global-hub-placementbinding-controller"
	config = rest.AddUserAgent(rest.CopyConfig(config), "global-hub-placementbinding")
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	informer, err := s.Cache.GetInformerForKind(ctx,
		policyv1.SchemeGroupVersion.WithKind("PlacementBinding"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		policyv1.SchemeGroupVersion.WithResource("placementbindings"), informer, s.Cache,
		func() client.Object { return &policyv1.PlacementBinding{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(
		hookContext genericapiserver.PostStartHookContext,
	) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}
