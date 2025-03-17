package syncers

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ensureMigrationClusterRole(client client.Client, ctx context.Context, logger *zap.SugaredLogger,
	msaName string) error {
	// create or update clusterrole for the migration service account
	migrationClusterRoleName := fmt.Sprintf("multicluster-global-hub-migration:%s", msaName)
	migrationClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: migrationClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
				Verbs:     []string{"create"},
			},
		},
	}

	foundMigrationClusterRole := &rbacv1.ClusterRole{}
	if err := client.Get(ctx,
		types.NamespacedName{
			Name: migrationClusterRoleName,
		}, foundMigrationClusterRole); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Infof("creating migration clusterrole %s", foundMigrationClusterRole.GetName())
			logger.Debugf("creating migration clusterrole %v", foundMigrationClusterRole)
			if err := client.Create(ctx, migrationClusterRole); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !apiequality.Semantic.DeepDerivative(migrationClusterRole, foundMigrationClusterRole) {
			logger.Infof("updating migration clusterrole %s", migrationClusterRole.GetName())
			logger.Debugf("updating migration clusterrole %v", migrationClusterRole)
			if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := client.Update(ctx, migrationClusterRole); err != nil {
					return err
				}
				return nil
			}); err != nil {
				logger.Errorw("failed to update migration ClusterRole", "error", err)
			}
		}
	}

	return nil
}

func ensureRegistrationClusterRoleBinding(client client.Client, ctx context.Context, logger *zap.SugaredLogger,
	msaName, msaNamespace string,
) error {
	registrationClusterRoleName := "open-cluster-management:managedcluster:bootstrap:agent-registration"
	registrationClusterRoleBindingName := fmt.Sprintf("agent-registration-clusterrolebinding:%s", msaName)
	registrationClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: registrationClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      msaName,
				Namespace: msaNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     registrationClusterRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	foundRegistrationClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	if err := client.Get(ctx,
		types.NamespacedName{
			Name: registrationClusterRoleBindingName,
		}, foundRegistrationClusterRoleBinding); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("creating agent registration clusterrolebinding",
				"clusterrolebinding", registrationClusterRoleBindingName)
			if err := client.Create(ctx, registrationClusterRoleBinding); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !apiequality.Semantic.DeepDerivative(registrationClusterRoleBinding, foundRegistrationClusterRoleBinding) {
			logger.Info("updating agent registration clusterrolebinding",
				"clusterrolebinding", registrationClusterRoleBindingName)
			if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := client.Update(ctx, registrationClusterRoleBinding); err != nil {
					return err
				}
				return nil
			}); err != nil {
				logger.Errorw("failed to update migration ClusterRoleBinding", "error", err)
			}
		}
	}

	return nil
}

func ensureSARClusterRoleBinding(client client.Client, ctx context.Context, logger *zap.SugaredLogger,
	msaName, msaNamespace string,
) error {
	migrationClusterRoleName := fmt.Sprintf("multicluster-global-hub-migration:%s", msaName)
	sarMigrationClusterRoleBindingName := fmt.Sprintf("%s-subjectaccessreviews-clusterrolebinding", msaName)
	sarMigrationClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: sarMigrationClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      msaName,
				Namespace: msaNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     migrationClusterRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	foundSAMigrationClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	if err := client.Get(ctx,
		types.NamespacedName{
			Name: sarMigrationClusterRoleBindingName,
		}, foundSAMigrationClusterRoleBinding); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Infof("creating subjectaccessreviews clusterrolebinding %s",
				foundSAMigrationClusterRoleBinding.GetName())
			logger.Debugf("creating subjectaccessreviews clusterrolebinding %v",
				foundSAMigrationClusterRoleBinding)
			if err := client.Create(ctx, sarMigrationClusterRoleBinding); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !apiequality.Semantic.DeepDerivative(sarMigrationClusterRoleBinding, foundSAMigrationClusterRoleBinding) {
			logger.Infof("updating subjectaccessreviews clusterrolebinding %v",
				sarMigrationClusterRoleBinding.GetName())
			logger.Debugf("updating subjectaccessreviews clusterrolebinding %v",
				sarMigrationClusterRoleBinding)
			if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				if err := client.Update(ctx, sarMigrationClusterRoleBinding); err != nil {
					return err
				}
				return nil
			}); err != nil {
				logger.Errorw("failed to update subjectaccessreviews ClusterRoleBinding", "error", err)
			}

		}
	}

	return nil
}

// createBookstrapKubeconfig creates a kubeconfig for the migration service account
// refer to https://github.com/open-cluster-management-io/ocm/blob/main/pkg/registration/register/common.go#L90-L101
// just need to refresh the token in case it is expired
func createBookstrapKubeconfig(client client.Client, ctx context.Context, restConfig *rest.Config,
	token string) (*clientcmdapi.Config, error) {

	kubeAPIServer, proxyURL, ca, caData, err := bootstrap.GetKubeAPIServerConfig(
		ctx, clientHolder, ns, mergedKlusterletConfig, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctxClusterName, err := bootstrap.GetKubeconfigClusterName(ctx, clientHolder.RuntimeClient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config := clientcmdapi.Config{
		// Define a cluster stanza based on the bootstrap kubeconfig.
		Clusters: map[string]*clientcmdapi.Cluster{
			ctxClusterName: {
				Server:                kubeAPIServer,
				InsecureSkipTLSVerify: false,
				// CertificateAuthority:     ca,
				CertificateAuthorityData: restConfig.CAData,
				// ProxyURL:                 proxyURL,
			}},
		// Define auth based on the obtained client cert.
		AuthInfos: map[string]*clientcmdapi.AuthInfo{"default-auth": {
			Token: string(token),
		}},
		// Define a context that connects the auth info and cluster, and set it as the default
		Contexts: map[string]*clientcmdapi.Context{"default-context": {
			Cluster:   ctxClusterName,
			AuthInfo:  "default-auth",
			Namespace: "default",
		}},
		CurrentContext: "default-context",
	}

	return config, nil
}
