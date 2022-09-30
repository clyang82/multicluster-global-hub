package apiserver

import (
	"context"

	"github.com/stolostron/multicluster-global-hub/manager/pkg/apiserver/etcd"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

type GlobalHubApiServer struct {
	postStartHooks   []postStartHookEntry
	preShutdownHooks []preShutdownHookEntry

	//contains server starting options
	options *Options

	hostedConfig *rest.Config
	// contains caches
	Cache cache.Cache

	client dynamic.Interface

	syncedCh chan struct{}
}

// postStartHookEntry groups a PostStartHookFunc with a name. We're not storing these hooks
// in a map and are instead letting the underlying api server perform the hook validation,
// such as checking for multiple PostStartHookFunc with the same name
type postStartHookEntry struct {
	name string
	hook genericapiserver.PostStartHookFunc
}

// preShutdownHookEntry fills the same purpose as postStartHookEntry except that it handles
// the PreShutdownHookFunc
type preShutdownHookEntry struct {
	name string
	hook genericapiserver.PreShutdownHookFunc
}

func NewGlobalHubApiServer(opts *Options, client dynamic.Interface,
	hostedConfig *rest.Config) *GlobalHubApiServer {
	return &GlobalHubApiServer{
		options:      opts,
		client:       client,
		hostedConfig: hostedConfig,
		syncedCh:     make(chan struct{}),
	}
}

// RunGlobalHubApiServer starts a new GlobalHubApiServer.
func (s *GlobalHubApiServer) RunGlobalHubApiServer(ctx context.Context) error {

	embeddedClientInfo, err := etcd.Run(context.TODO(), "2380", "2379")
	if err != nil {
		return err
	}

	genericConfig, genericEtcdOptions, extensionServer, err := CreateExtensions(s.options, embeddedClientInfo)
	if err != nil {
		return err
	}

	config, err := CreateAggregatorConfig(genericConfig, genericEtcdOptions)
	if err != nil {
		return err
	}

	aggregatorServer, err := CreateAggregatorServer(config,
		extensionServer.GenericAPIServer, extensionServer.Informers)
	if err != nil {
		return err
	}

	err = s.CreateCache(ctx)
	if err != nil {
		return err
	}

	controllerConfig := rest.CopyConfig(aggregatorServer.GenericAPIServer.LoopbackClientConfig)

	err = s.InstallCRDController(ctx, controllerConfig)
	if err != nil {
		return err
	}

	// err = s.InstallPolicyController(ctx, controllerConfig)
	// if err != nil {
	// 	return err
	// }

	// err = s.InstallPlacementRuleController(ctx, controllerConfig)
	// if err != nil {
	// 	return err
	// }

	// err = s.InstallPlacementBindingController(ctx, controllerConfig)
	// if err != nil {
	// 	return err
	// }

	// TODO: kubectl explain currently failing on crd resources, but works on apiservices
	// kubectl get and describe do work, though

	// Add our custom hooks to the underlying api server
	for _, entry := range s.postStartHooks {
		err := aggregatorServer.GenericAPIServer.AddPostStartHook(entry.name, entry.hook)
		if err != nil {
			return err
		}
	}

	return RunAggregator(aggregatorServer, ctx.Done())
}

// AddPostStartHook allows you to add a PostStartHook that gets passed to the underlying genericapiserver implementation.
func (s *GlobalHubApiServer) AddPostStartHook(name string, hook genericapiserver.PostStartHookFunc) {
	// you could potentially add duplicate or invalid post start hooks here, but we'll let
	// the genericapiserver implementation do its own validation during startup.
	s.postStartHooks = append(s.postStartHooks, postStartHookEntry{
		name: name,
		hook: hook,
	})
}

// AddPreShutdownHook allows you to add a PreShutdownHookFunc that gets passed to the underlying genericapiserver implementation.
func (s *GlobalHubApiServer) AddPreShutdownHook(name string, hook genericapiserver.PreShutdownHookFunc) {
	// you could potentially add duplicate or invalid post start hooks here, but we'll let
	// the genericapiserver implementation do its own validation during startup.
	s.preShutdownHooks = append(s.preShutdownHooks, preShutdownHookEntry{
		name: name,
		hook: hook,
	})
}

// func (s *GlobalHubApiServer) waitForSync(stop <-chan struct{}) error {
// 	// Wait for shared informer factories to by synced.
// 	// factory. Otherwise, informer list calls may go into backoff (before the CRDs are ready) and
// 	// take ~10 seconds to succeed.
// 	select {
// 	case <-stop:
// 		return errors.New("timed out waiting for informers to sync")
// 	case <-s.syncedCh:
// 		return nil
// 	}
// }
