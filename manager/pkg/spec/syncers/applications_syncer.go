package syncers

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/stolostron/multicluster-global-hub/manager/pkg/spec/controllers/bundle"
	"github.com/stolostron/multicluster-global-hub/manager/pkg/spec/specdb"
	"github.com/stolostron/multicluster-global-hub/manager/pkg/spec/syncers/interval"
	"github.com/stolostron/multicluster-global-hub/pkg/logger"
	"github.com/stolostron/multicluster-global-hub/pkg/transport"
)

const (
	applicationsTableName = "applications"
	applicationsMsgKey    = "Applications"
)

// AddApplicationsDBToTransportSyncer adds applications db to transport syncer to the manager.
func AddApplicationsDBToTransportSyncer(mgr ctrl.Manager, specDB specdb.SpecDB, producer transport.Producer,
	specSyncInterval time.Duration,
) error {
	createObjFunc := func() metav1.Object { return &applicationv1beta1.Application{} }
	lastSyncTimestampPtr := &time.Time{}

	if err := mgr.Add(&genericDBToTransportSyncer{
		log:            logger.ZapLogger("db-to-transport-syncer-application"),
		intervalPolicy: interval.NewExponentialBackoffPolicy(specSyncInterval),
		syncBundleFunc: func(ctx context.Context) (bool, error) {
			return syncObjectsBundle(ctx, producer, applicationsMsgKey, specDB, applicationsTableName,
				createObjFunc, bundle.NewBaseObjectsBundle, lastSyncTimestampPtr)
		},
	}); err != nil {
		return fmt.Errorf("failed to add applications db to transport syncer - %w", err)
	}

	return nil
}
