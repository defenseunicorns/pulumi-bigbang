package v2

import (
	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
)

const DEFUALT_GATEKEEPER_VERSION = "3.5.2-bb.0"

var (
	oci_gatekeeper_registry    = "oci://registry.dso.mil/platform-one/big-bang/bigbang/gatekeeper"
	gatekeeper_version         = "*"
	defaultGatekeeperNamespace = "gatekeeper-system"
)

type Gatekeeper struct {
	Resources []pulumi.Resource
}

func (g Gatekeeper) NetworkPolicies() []string {
	return make([]string, 0)
}

func (g Gatekeeper) Enabled() bool { return true }

func (g Gatekeeper) GetResources() ([]pulumi.Resource, error) { return g.Resources, nil }

func (g Gatekeeper) GetViolations() *api.Violations {
	return &api.Violations{}
}

func (g Gatekeeper) Deploy(ctx *pulumi.Context, bb api.BigBang, deps ...pulumi.Resource) ([]pulumi.Resource, error) {
	//Create Namespace
	var namespace = defaultGatekeeperNamespace
	// if bb.Configuration.Policy.CommonConfig.Namespace != "" {
	// 	namespace = bb.Configuration.Policy.CommonConfig.Namespace
	// }

	//might have to set some context things or

	var pc PullCreds

	if len(bb.Configuration.ImagePullSecrets) > 0 {
		pc.Username = bb.Configuration.ImagePullSecrets[0].Username
		pc.Password = bb.Configuration.ImagePullSecrets[0].Password
		pc.Registry = bb.Configuration.ImagePullSecrets[0].Registry
	}

	DeployNamespace(ctx, namespace, bb.Configuration.ServiceMesh.Name != api.ServiceMeshNone, pc)

	//Prep the gatekeeper values based on what's been deployed

	values := make(pulumi.Map)

	// This first set could/should be standard across apps.
	values["networkPolicies"] = pulumi.Map{
		"enabled":          pulumi.Bool(bb.Configuration.NetworkPolicies.Enabled),
		"controlPlaneCidr": pulumi.String(bb.Configuration.NetworkPolicies.ControlPlaneCIDR),
	}

	values["serviceMonitor"] = pulumi.Map{
		"enabled": pulumi.Bool(bb.Configuration.Monitoring.Name != api.MonitoringNone),
	}
	dockerRegistries := make([]string, 0)
	hostFilesystem := make([]string, 0)
	hostNetworkNamespcae := make([]string, 0)
	for _, p := range bb.Packages {
		violations := p.GetViolations()

		dockerRegistries = append(dockerRegistries, violations.AllowedDockerRegistries...)
		hostFilesystem = append(hostFilesystem, violations.AllowedHostFilesystem...)
		hostNetworkNamespcae = append(hostNetworkNamespcae, violations.NoHostNamespace...)

	}

	if bb.Configuration.Development {
		dockerRegistries = append(dockerRegistries, "rancher/klipper-lb")
	}

	pullSecretMap := pulumi.MapArray{
		pulumi.Map{
			"name": pulumi.String("private-registry"),
		},
	}
	imageMap := pulumi.Map{
		"pullSecrets": pullSecretMap,
	}

	// Add IPS to config
	values["image"] = pulumi.Map{
		"pullSecrets": pullSecretMap,
	}
	values["postInstall"] = pulumi.Map{
		"labelNamespace": pulumi.Map{
			"image": imageMap,
		},
		"probeWebhook": pulumi.Map{
			"image": imageMap,
		},
	}
	values["postUpgrade"] = pulumi.Map{
		"cleanupCRD": pulumi.Map{
			"image": imageMap,
		},
	}
	values["preUninstall"] = pulumi.Map{
		"deleteWebhookConfigurations": pulumi.Map{
			"image": imageMap,
		},
	}

	values["violations"] = pulumi.Map{
		"allowedDockerRegistries": pulumi.Map{
			"parameters": pulumi.Map{
				"repos": pulumi.ToStringArray(dockerRegistries),
			},
		},
		"allowedHostFilesystem": pulumi.Map{
			"parameters": pulumi.Map{
				"excludedResources": pulumi.ToStringArray(hostFilesystem),
			},
		},
		"hostNetworking": pulumi.Map{
			"parameters": pulumi.Map{
				"excludedResources": pulumi.ToStringArray(hostNetworkNamespcae),
			},
		},
		"namespacesHaveIstio": pulumi.Map{
			"enabled": pulumi.Bool(bb.Configuration.ServiceMesh.Name != api.ServiceMeshNone),
		},
		"podsHaveIstio": pulumi.Map{
			"parameters": pulumi.Map{
				"excludedNamespaces": pulumi.ToStringArray([]string{
					"istio-operator",
					"istio-system",
				}),
			},
		},
	}

	releaseArgs := &helmv3.ReleaseArgs{
		Chart:   pulumi.String(oci_gatekeeper_registry),
		Version: pulumi.String(DEFUALT_GATEKEEPER_VERSION),
		// RepositoryOpts: helmv3.RepositoryOptsArgs{
		// 	Repo: pulumi.String(`oci://registry.dso.mil/platform-one/big-bang/bigbang/`),
		// },
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(namespace),
		Name:            pulumi.String("gatekeeper"),
		// ValueYamlFiles: pulumi.NewAssetArchive(map[string]interface{}{
		// 	"certs": pulumi.NewFileAsset("https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/ingress-certs.yaml"),
		// }),
		Values: values,
	}
	// Deploy v9.6.0 version of the wordpress chart.
	release, err := helmv3.NewRelease(ctx, "gatekeeper", releaseArgs, pulumi.DependsOn(deps))
	//  helm install bigbang --create-namespace oci://registry.dso.mil/platform-one/big-bang/bigbang/bigbang --version 1.35.0 -n bigbang -f ./chart/ingress-certs.yaml -f dev/credentials.yaml

	//TODO do something better here

	return []pulumi.Resource{release}, err

}
