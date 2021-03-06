package bootstrappolicy

import (
	"strings"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"

	authorizationapi "github.com/openshift/origin/pkg/authorization/apis/authorization"
)

const saRolePrefix = "system:openshift:controller:"

var (
	// controllerRoles is a slice of roles used for controllers
	controllerRoles = []rbac.ClusterRole{}
	// controllerRoleBindings is a slice of roles used for controllers
	controllerRoleBindings = []rbac.ClusterRoleBinding{}
)

func addControllerRole(role rbac.ClusterRole) {
	if !strings.HasPrefix(role.Name, saRolePrefix) {
		glog.Fatalf(`role %q must start with %q`, role.Name, saRolePrefix)
	}

	for _, existingRole := range controllerRoles {
		if role.Name == existingRole.Name {
			glog.Fatalf("role %q was already registered", role.Name)
		}
	}

	if role.Annotations == nil {
		role.Annotations = map[string]string{}
	}
	role.Annotations[roleSystemOnly] = roleIsSystemOnly

	controllerRoles = append(controllerRoles, role)

	controllerRoleBindings = append(controllerRoleBindings,
		rbac.NewClusterBinding(role.Name).SAs(DefaultOpenShiftInfraNamespace, role.Name[len(saRolePrefix):]).BindingOrDie())
}

func eventsRule() rbac.PolicyRule {
	return rbac.NewRule("create", "update", "patch").Groups(kapiGroup).Resources("events").RuleOrDie()
}

func init() {
	// build-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraBuildControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch", "patch", "update", "delete").Groups(buildGroup, legacyBuildGroup).Resources("builds").RuleOrDie(),
			rbac.NewRule("get").Groups(buildGroup, legacyBuildGroup).Resources("buildconfigs").RuleOrDie(),
			rbac.NewRule("create").Groups(buildGroup, legacyBuildGroup).Resources("builds/optimizeddocker", "builds/docker", "builds/source", "builds/custom", "builds/jenkinspipeline").RuleOrDie(),
			rbac.NewRule("get", "list").Groups(imageGroup, legacyImageGroup).Resources("imagestreams").RuleOrDie(),
			rbac.NewRule("get", "list").Groups(kapiGroup).Resources("secrets").RuleOrDie(),
			rbac.NewRule("get", "list").Groups(kapiGroup).Resources("configmaps").RuleOrDie(),
			rbac.NewRule("get", "list", "create", "delete").Groups(kapiGroup).Resources("pods").RuleOrDie(),
			rbac.NewRule("get").Groups(kapiGroup).Resources("namespaces").RuleOrDie(),
			eventsRule(),
		},
	})

	// build-config-change-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraBuildConfigChangeControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch").Groups(buildGroup, legacyBuildGroup).Resources("buildconfigs").RuleOrDie(),
			rbac.NewRule("create").Groups(buildGroup, legacyBuildGroup).Resources("buildconfigs/instantiate").RuleOrDie(),
			eventsRule(),
		},
	})

	// deployer-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraDeployerControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("create", "get", "list", "watch", "patch", "delete").Groups(kapiGroup).Resources("pods").RuleOrDie(),

			// "delete" is required here for compatibility with older deployer images
			// (see https://github.com/openshift/origin/pull/14322#issuecomment-303968976)
			// TODO: remove "delete" rule few releases after 3.6
			rbac.NewRule("delete").Groups(kapiGroup).Resources("replicationcontrollers").RuleOrDie(),
			rbac.NewRule("get", "list", "watch", "update").Groups(kapiGroup).Resources("replicationcontrollers").RuleOrDie(),
			eventsRule(),
		},
	})

	// deploymentconfig-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraDeploymentConfigControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("create", "get", "list", "watch", "update", "patch", "delete").Groups(kapiGroup).Resources("replicationcontrollers").RuleOrDie(),
			rbac.NewRule("update").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs/status").RuleOrDie(),
			rbac.NewRule("get", "list", "watch").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs").RuleOrDie(),
			eventsRule(),
		},
	})

	// deployment-trigger-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraDeploymentTriggerControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch").Groups(kapiGroup).Resources("replicationcontrollers").RuleOrDie(),
			rbac.NewRule("get", "list", "watch").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs").RuleOrDie(),
			rbac.NewRule("get", "list", "watch").Groups(imageGroup, legacyImageGroup).Resources("imagestreams").RuleOrDie(),

			rbac.NewRule("create").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs/instantiate").RuleOrDie(),
			eventsRule(),
		},
	})

	// template-instance-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraTemplateInstanceControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("create").Groups(kAuthzGroup).Resources("subjectaccessreviews").RuleOrDie(),
			rbac.NewRule("get", "list", "watch").Groups(templateGroup).Resources("subjectaccessreviews").RuleOrDie(),
			rbac.NewRule("update").Groups(templateGroup).Resources("templateinstances/status").RuleOrDie(),
		},
	})

	// template-instance-controller
	controllerRoleBindings = append(controllerRoleBindings,
		rbac.NewClusterBinding(AdminRoleName).SAs(DefaultOpenShiftInfraNamespace, InfraTemplateInstanceControllerServiceAccountName).BindingOrDie())

	// origin-namespace-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraOriginNamespaceServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch").Groups(kapiGroup).Resources("namespaces").RuleOrDie(),
			rbac.NewRule("update").Groups(kapiGroup).Resources("namespaces/finalize", "namespaces/status").RuleOrDie(),
			eventsRule(),
		},
	})

	// serviceaccount-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraServiceAccountControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch", "create", "update", "patch", "delete").Groups(kapiGroup).Resources("serviceaccounts").RuleOrDie(),
			eventsRule(),
		},
	})

	// serviceaccount-pull-secrets-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraServiceAccountPullSecretsControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch", "create", "update").Groups(kapiGroup).Resources("serviceaccounts").RuleOrDie(),
			rbac.NewRule("get", "list", "watch", "create", "update", "patch", "delete").Groups(kapiGroup).Resources("secrets").RuleOrDie(),
			rbac.NewRule("get", "list", "watch").Groups(kapiGroup).Resources("services").RuleOrDie(),
			eventsRule(),
		},
	})

	// imagetrigger-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraImageTriggerControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("list", "watch").Groups(imageGroup, legacyImageGroup).Resources("imagestreams").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(extensionsGroup).Resources("daemonsets").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(extensionsGroup, appsGroup).Resources("deployments").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(appsGroup).Resources("statefulsets").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(batchGroup).Resources("cronjobs").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs").RuleOrDie(),
			rbac.NewRule("create").Groups(buildGroup, legacyBuildGroup).Resources("buildconfigs/instantiate").RuleOrDie(),
			// trigger controller must be able to modify these build types
			// TODO: move to a new custom binding that can be removed separately from end user access?
			rbac.NewRule("create").Groups(buildGroup, legacyBuildGroup).Resources(
				authorizationapi.SourceBuildResource,
				authorizationapi.DockerBuildResource,
				authorizationapi.OptimizedDockerBuildResource,
				authorizationapi.JenkinsPipelineBuildResource,
			).RuleOrDie(),

			eventsRule(),
		},
	})

	// service-serving-cert-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraServiceServingCertServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("list", "watch", "update").Groups(kapiGroup).Resources("services").RuleOrDie(),
			rbac.NewRule("get", "list", "watch", "create", "update").Groups(kapiGroup).Resources("secrets").RuleOrDie(),
			eventsRule(),
		},
	})

	// image-import-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraImageImportControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list", "watch", "create", "update").Groups(imageGroup, legacyImageGroup).Resources("imagestreams").RuleOrDie(),
			rbac.NewRule("get", "list", "watch", "create", "update", "patch", "delete").Groups(imageGroup, legacyImageGroup).Resources("images").RuleOrDie(),
			rbac.NewRule("create").Groups(imageGroup, legacyImageGroup).Resources("imagestreamimports").RuleOrDie(),
			eventsRule(),
		},
	})

	// sdn-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraSDNControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "create", "update").Groups(networkGroup, legacyNetworkGroup).Resources("clusternetworks").RuleOrDie(),
			rbac.NewRule("get", "list").Groups(networkGroup, legacyNetworkGroup).Resources("egressnetworkpolicies").RuleOrDie(),
			rbac.NewRule("get", "list", "create", "delete").Groups(networkGroup, legacyNetworkGroup).Resources("hostsubnets").RuleOrDie(),
			rbac.NewRule("get", "list", "create", "update", "delete").Groups(networkGroup, legacyNetworkGroup).Resources("netnamespaces").RuleOrDie(),
			rbac.NewRule("get", "list").Groups(kapiGroup).Resources("pods").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("services").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("namespaces").RuleOrDie(),
			rbac.NewRule("get").Groups(kapiGroup).Resources("nodes").RuleOrDie(),
			rbac.NewRule("update").Groups(kapiGroup).Resources("nodes/status").RuleOrDie(),
			rbac.NewRule("list").Groups(extensionsGroup).Resources("networkPolicies").RuleOrDie(),

			eventsRule(),
		},
	})

	// cluster-quota-reconciliation
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraClusterQuotaReconciliationControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "list").Groups(kapiGroup).Resources("configMaps").RuleOrDie(),
			rbac.NewRule("get", "list").Groups(kapiGroup).Resources("secrets").RuleOrDie(),
			rbac.NewRule("update").Groups(quotaGroup, legacyQuotaGroup).Resources("clusterresourcequotas/status").RuleOrDie(),
			eventsRule(),
		},
	})

	// unidling-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraUnidlingControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "update").Groups(kapiGroup).Resources("replicationcontrollers/scale", "endpoints").RuleOrDie(),
			rbac.NewRule("get", "update", "patch").Groups(kapiGroup).Resources("replicationcontrollers").RuleOrDie(),
			rbac.NewRule("get", "update", "patch").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(extensionsGroup, appsGroup).Resources("replicasets/scale", "deployments/scale").RuleOrDie(),
			rbac.NewRule("get", "update").Groups(deployGroup, legacyDeployGroup).Resources("deploymentconfigs/scale").RuleOrDie(),
			rbac.NewRule("watch", "list").Groups(kapiGroup).Resources("events").RuleOrDie(),
			eventsRule(),
		},
	})

	// ingress-ip-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraServiceIngressIPControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("list", "watch", "update").Groups(kapiGroup).Resources("services").RuleOrDie(),
			rbac.NewRule("update").Groups(kapiGroup).Resources("services/status").RuleOrDie(),
			eventsRule(),
		},
	})

	// pv-recycler-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraPersistentVolumeRecyclerControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("get", "update", "create", "delete", "list", "watch").Groups(kapiGroup).Resources("persistentvolumes").RuleOrDie(),
			rbac.NewRule("update").Groups(kapiGroup).Resources("persistentvolumes/status").RuleOrDie(),
			rbac.NewRule("get", "update", "list", "watch").Groups(kapiGroup).Resources("persistentvolumeclaims").RuleOrDie(),
			rbac.NewRule("update").Groups(kapiGroup).Resources("persistentvolumeclaims/status").RuleOrDie(),
			rbac.NewRule("get", "create", "delete", "list", "watch").Groups(kapiGroup).Resources("pods").RuleOrDie(),
			eventsRule(),
		},
	})

	// resourcequota-controller
	addControllerRole(rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: saRolePrefix + InfraResourceQuotaControllerServiceAccountName},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("update").Groups(kapiGroup).Resources("resourcequotas/status").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("resourcequotas").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("services").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("configmaps").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("secrets").RuleOrDie(),
			rbac.NewRule("list").Groups(kapiGroup).Resources("replicationcontrollers").RuleOrDie(),
			eventsRule(),
		},
	})

}

// ControllerRoles returns the cluster roles used by controllers
func ControllerRoles() []rbac.ClusterRole {
	return controllerRoles
}

// ControllerRoleBindings returns the role bindings used by controllers
func ControllerRoleBindings() []rbac.ClusterRoleBinding {
	return controllerRoleBindings
}
