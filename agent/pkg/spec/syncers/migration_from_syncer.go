// Copyright (c) 2024 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package syncers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	klusterletv1alpha1 "github.com/stolostron/cluster-lifecycle-api/klusterletconfig/v1alpha1"
	addonv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bundleevent "github.com/stolostron/multicluster-global-hub/pkg/bundle/event"
	eventversion "github.com/stolostron/multicluster-global-hub/pkg/bundle/version"
	"github.com/stolostron/multicluster-global-hub/pkg/constants"
	"github.com/stolostron/multicluster-global-hub/pkg/enum"
	"github.com/stolostron/multicluster-global-hub/pkg/logger"
	"github.com/stolostron/multicluster-global-hub/pkg/transport"
)

// This is a temporary solution to wait for applying the klusterletconfig
var sleepForApplying = 20 * time.Second

type managedClusterMigrationFromSyncer struct {
	log             *zap.SugaredLogger
	client          client.Client
	transportClient transport.TransportClient
	config          *rest.Config
	nativeClient    *kubernetes.Clientset
}

func NewManagedClusterMigrationFromSyncer(client client.Client,
	transportClient transport.TransportClient,
) (*managedClusterMigrationFromSyncer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	nativeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &managedClusterMigrationFromSyncer{
		log:             logger.ZapLogger("managed-cluster-migration-from-syncer"),
		client:          client,
		transportClient: transportClient,
		config:          config,
		nativeClient:    nativeClient,
	}, nil
}

func (s *managedClusterMigrationFromSyncer) Sync(ctx context.Context, payload []byte) error {
	// handle migration.from cloud event
	managedClusterMigrationEvent := &bundleevent.ManagedClusterMigrationFromEvent{}
	if err := json.Unmarshal(payload, managedClusterMigrationEvent); err != nil {
		return err
	}
	s.log.Debugf("received managed cluster migration event %s", string(payload))

	// create or update bootstrap secret
	bootstrapSecret := managedClusterMigrationEvent.BootstrapSecret
	foundBootstrapSecret := &corev1.Secret{}
	if err := s.client.Get(ctx,
		types.NamespacedName{
			Name:      bootstrapSecret.Name,
			Namespace: bootstrapSecret.Namespace,
		}, foundBootstrapSecret); err != nil {
		if apierrors.IsNotFound(err) {
			s.log.Infof("creating bootstrap secret %s", bootstrapSecret.GetName())
			if err := s.client.Create(ctx, bootstrapSecret); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// update the bootstrap secret if it already exists
		s.log.Infof("updating bootstrap secret %s", bootstrapSecret.GetName())
		if err := s.client.Update(ctx, bootstrapSecret); err != nil {
			return err
		}
	}

	// create a current hub kubeconfig for the managed cluster
	if err := s.createBootstrapKubeconfigForCurrentHub(ctx); err != nil {
		return err
	}

	// create klusterlet config if it does not exist
	// TODO: do not need to transfer from global hub. it can be created in this cluster
	// TODO: update the klusterlet config if it already exists
	klusterletConfig := managedClusterMigrationEvent.KlusterletConfig
	// set the bootstrap kubeconfig secrets in klusterlet config
	klusterletConfig.Spec.BootstrapKubeConfigs.LocalSecrets.KubeConfigSecrets = []operatorv1.KubeConfigSecret{
		{
			Name: bootstrapSecret.Name,
		},
	}
	foundKlusterletConfig := &klusterletv1alpha1.KlusterletConfig{}
	if err := s.client.Get(ctx,
		types.NamespacedName{
			Name: klusterletConfig.Name,
		}, foundKlusterletConfig); err != nil {
		if apierrors.IsNotFound(err) {
			s.log.Infof("creating klusterlet config %s", klusterletConfig.GetName())
			s.log.Debugf("creating klusterlet config %v", klusterletConfig)
			if err := s.client.Create(ctx, klusterletConfig); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	managedClusters := managedClusterMigrationEvent.ManagedClusters
	// update managed cluster annotations to point to the new klusterlet config
	for _, managedCluster := range managedClusters {
		mc := &clusterv1.ManagedCluster{}
		if err := s.client.Get(ctx, types.NamespacedName{
			Name: managedCluster,
		}, mc); err != nil {
			return err
		}
		annotations := mc.Annotations
		if annotations == nil {
			annotations = make(map[string]string)
		}

		_, migrating := annotations[constants.ManagedClusterMigrating]
		if migrating && annotations["agent.open-cluster-management.io/klusterlet-config"] == klusterletConfig.Name {
			continue
		}
		annotations["agent.open-cluster-management.io/klusterlet-config"] = klusterletConfig.Name
		annotations[constants.ManagedClusterMigrating] = ""
		mc.SetAnnotations(annotations)
		if err := s.client.Update(ctx, mc); err != nil {
			return err
		}
	}

	// send KlusterletAddonConfig to the global hub and then propogate to the target cluster
	for _, managedCluster := range managedClusters {
		if err := s.sendKlusterletAddonConfig(ctx, managedCluster); err != nil {
			return err
		}
	}

	// wait for 10 seconds to ensure the klusterletconfig is applied and then trigger the migration
	// right now, no condition indicates the klusterletconfig is applied
	time.Sleep(sleepForApplying)
	for _, managedCluster := range managedClusters {
		mc := &clusterv1.ManagedCluster{}
		if err := s.client.Get(ctx, types.NamespacedName{
			Name: managedCluster,
		}, mc); err != nil {
			return err
		}
		mc.Spec.HubAcceptsClient = false
		s.log.Infof("updating managedcluster %s to set HubAcceptsClient as false", mc.Name)
		if err := s.client.Update(ctx, mc); err != nil {
			return err
		}
	}

	time.Sleep(sleepForApplying)
	if err := s.detachManagedClusters(ctx, managedClusters); err != nil {
		s.log.Error(err, "failed to detach managed clusters")
		return err
	}

	return nil
}

// createNewBootstrapKubeconfigForCurrentHub creates kubeconfig for the current hub due to:
// 1. multiple hub feature cannot support only one kubeconfigSecret in klusterletconfig
// 2. without current hub kubeconfig, the managed cluster will be switched to the new hub at once. it may lead to too many CSRs issue.
// Cannot use the bootstrap-hub-kubeconfig in klusterlet manifestwork, because the bootstrap-hub-kubeconfig cannot be expired.
// So we need to generate a new kubeconfig for the current hub
// in this way, it does not depend on the managed service account. it can be beneficial for standalone global hub agent case.
func (s *managedClusterMigrationFromSyncer) createBootstrapKubeconfigForCurrentHub(ctx context.Context) error {
	// create a service account in global hub agent namespace
	if err := s.ensureServiceAccount(ctx); err != nil {
		return err
	}
	// create sar clusterrole to the service account

	if err := ensureMigrationClusterRole(s.client, ctx, s.log, constants.MigrationFromServiceAccountName); err != nil {
		return err
	}
	// create a clusterrolebinding for the service account

	if err := ensureRegistrationClusterRoleBinding(s.client, ctx, s.log, constants.MigrationFromServiceAccountName,
		constants.GHAgentNamespace); err != nil {
		return err
	}

	if err := ensureSARClusterRoleBinding(s.client, ctx, s.log, constants.MigrationFromServiceAccountName,
		constants.GHAgentNamespace); err != nil {
		return err
	}

	// create a token for the service account
	token, err := s.createTokenByTokenRequest(ctx)
	if err != nil {
		return err
	}

	// create a kubeconfig based on the service account token

	return nil
}

// cleanupMigrationFromResources cleans up the resources created in the migration from cluster
// TODO: implement the cleanup logic
func cleanupMigrationFromResources(ctx context.Context, client client.Client, log *zap.SugaredLogger) error {
	return nil
}

// sendKlusterletAddonConfig sends the klusterletAddonConfig back to the global hub
func (s *managedClusterMigrationFromSyncer) sendKlusterletAddonConfig(ctx context.Context,
	managedCluster string,
) error {
	config := &addonv1.KlusterletAddonConfig{}
	// send klusterletAddonConfig to global hub so that it can be transferred to the target cluster
	if err := s.client.Get(ctx, types.NamespacedName{
		Name:      managedCluster,
		Namespace: managedCluster,
	}, config); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	// do cleanup
	config.SetManagedFields(nil)
	config.SetFinalizers(nil)
	config.SetOwnerReferences(nil)
	config.SetSelfLink("")
	config.SetResourceVersion("")
	config.SetGeneration(0)
	config.Status = addonv1.KlusterletAddonConfigStatus{}

	payloadBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal klusterletAddonConfig (%v) - %w", config, err)
	}

	version := eventversion.NewVersion()
	version.Incr() // first generation -> reset
	e := cloudevents.NewEvent()
	e.SetType(string(enum.KlusterletAddonConfigType))
	e.SetSource(constants.CloudEventSourceGlobalHub)
	e.SetExtension(eventversion.ExtVersion, version.String())
	_ = e.SetData(cloudevents.ApplicationJSON, payloadBytes)
	if s.transportClient != nil {
		if err := s.transportClient.GetProducer().SendEvent(ctx, e); err != nil {
			return fmt.Errorf("failed to send klusterletAddonConfig back to the global hub, due to %v", err)
		}
	}
	return nil
}

func (s *managedClusterMigrationFromSyncer) detachManagedClusters(ctx context.Context, managedClusters []string) error {
	for _, managedCluster := range managedClusters {
		mc := &clusterv1.ManagedCluster{}
		if err := s.client.Get(ctx, types.NamespacedName{
			Name: managedCluster,
		}, mc); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			} else {
				return err
			}
		}
		if !mc.Spec.HubAcceptsClient {
			if err := s.client.Delete(ctx, mc); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *managedClusterMigrationFromSyncer) ensureServiceAccount(ctx context.Context) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.GHAgentNamespace,
			Name:      constants.MigrationFromServiceAccountName,
		},
	}
	if err := r.client.Create(ctx, sa); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (r *managedClusterMigrationFromSyncer) createTokenByTokenRequest(ctx context.Context) (string, error) {
	tr, err := r.nativeClient.CoreV1().ServiceAccounts(constants.GHAgentNamespace).
		CreateToken(context.TODO(), constants.MigrationFromServiceAccountName, &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{},
		}, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return tr.Status.Token, nil
}
