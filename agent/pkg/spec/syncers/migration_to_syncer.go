// Copyright (c) 2024 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package syncers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bundleevent "github.com/stolostron/multicluster-global-hub/pkg/bundle/event"
	"github.com/stolostron/multicluster-global-hub/pkg/logger"
)

type managedClusterMigrationToSyncer struct {
	log    *zap.SugaredLogger
	client client.Client
}

func NewManagedClusterMigrationToSyncer(client client.Client) *managedClusterMigrationToSyncer {
	return &managedClusterMigrationToSyncer{
		log:    logger.ZapLogger("managed-cluster-migration-to-syncer"),
		client: client,
	}
}

func (s *managedClusterMigrationToSyncer) Sync(ctx context.Context, payload []byte) error {
	// handle migration.to cloud event
	s.log.Info("received cloudevent from the global hub")
	managedClusterMigrationToEvent := &bundleevent.ManagedClusterMigrationToEvent{}
	if err := json.Unmarshal(payload, managedClusterMigrationToEvent); err != nil {
		return err
	}
	s.log.Debugf("received cloudevent %v", string(payload))

	msaName := managedClusterMigrationToEvent.ManagedServiceAccountName
	msaNamespace := managedClusterMigrationToEvent.ManagedServiceAccountInstallNamespace

	if err := s.ensureClusterManager(ctx, msaName, msaNamespace); err != nil {
		return err
	}

	if err := ensureMigrationClusterRole(s.client, ctx, s.log, msaName); err != nil {
		return err
	}

	if err := ensureRegistrationClusterRoleBinding(s.client, ctx, s.log, msaName, msaNamespace); err != nil {
		return err
	}

	if err := ensureSARClusterRoleBinding(s.client, ctx, s.log, msaName, msaNamespace); err != nil {
		return err
	}

	klusterletAddonConfig := managedClusterMigrationToEvent.KlusterletAddonConfig
	if klusterletAddonConfig != nil {
		return wait.PollUntilContextTimeout(ctx, 1*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			if err := s.client.Create(ctx, klusterletAddonConfig); err != nil {
				s.log.Debugf("cannot create klusterletAddonConfig %v", err)
				return false, nil
			}
			return true, nil
		})
	}
	return nil
}

func (s *managedClusterMigrationToSyncer) ensureClusterManager(ctx context.Context,
	msaName, msaNamespace string,
) error {
	foundClusterManager := &operatorv1.ClusterManager{}
	if err := s.client.Get(ctx,
		types.NamespacedName{Name: "cluster-manager"}, foundClusterManager); err != nil {
		return err
	}

	clusterManager := foundClusterManager.DeepCopy()
	// check if the ManagedClusterAutoApproval feature is enabled and
	// the service account is added to the auto-approve list
	autoApproveUser := fmt.Sprintf("system:serviceaccount:%s:%s", msaNamespace, msaName)
	autoApproveFeatureEnabled := false
	autoApproveUserAdded := false
	clusterManagerChanged := false
	if clusterManager.Spec.RegistrationConfiguration != nil {
		registrationConfiguration := clusterManager.Spec.RegistrationConfiguration
		for i, featureGate := range registrationConfiguration.FeatureGates {
			if featureGate.Feature == "ManagedClusterAutoApproval" {
				autoApproveFeatureEnabled = true
				if featureGate.Mode == operatorv1.FeatureGateModeTypeEnable {
					break
				} else {
					registrationConfiguration.FeatureGates[i].Mode = operatorv1.FeatureGateModeTypeEnable
					clusterManagerChanged = true
					break
				}
			}
		}
		if !autoApproveFeatureEnabled {
			registrationConfiguration.FeatureGates = append(
				registrationConfiguration.FeatureGates,
				operatorv1.FeatureGate{
					Feature: "ManagedClusterAutoApproval",
					Mode:    operatorv1.FeatureGateModeTypeEnable,
				})
			clusterManagerChanged = true
		}
		for _, user := range registrationConfiguration.AutoApproveUsers {
			if user == autoApproveUser {
				autoApproveUserAdded = true
				break
			}
		}
		if !autoApproveUserAdded {
			registrationConfiguration.AutoApproveUsers = append(
				registrationConfiguration.AutoApproveUsers,
				autoApproveUser)
			clusterManagerChanged = true
		}
	} else {
		clusterManager.Spec.RegistrationConfiguration = &operatorv1.RegistrationHubConfiguration{
			FeatureGates: []operatorv1.FeatureGate{
				{
					Feature: "ManagedClusterAutoApproval",
					Mode:    operatorv1.FeatureGateModeTypeEnable,
				},
			},
			AutoApproveUsers: []string{
				autoApproveUser,
			},
		}
		clusterManagerChanged = true
	}

	// patch cluster-manager only if it has changed
	if clusterManagerChanged {
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			if err := s.client.Update(ctx, clusterManager); err != nil {
				return err
			}
			return nil
		}); err != nil {
			s.log.Errorw("failed to update clusterManager", "error", err)
		}
	}

	return nil
}
