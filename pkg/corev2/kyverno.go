package v2

import (
	"strings"

	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
)

const DEFUALT_KYVERNO_VERSION = "2.5.2-bb.0"
const DEFAULT_KYVERNO_POLICY_VERSION = "1.0.1-bb.1"

var (
	oci_kyverno_registry          = "oci://registry.dso.mil/platform-one/big-bang/bigbang/kyverno"
	oci_kyverno_policies_policies = "oci://registry.dso.mil/platform-one/big-bang/bigbang/kyverno-policies"
	version                       = "*"
	defaultKyvernoNamespace       = "kyverno"
)

type Kyverno struct {
	Resources []pulumi.Resource
	BigBang   api.BigBang
}

func (g Kyverno) NetworkPolicies() []string {
	return make([]string, 0)
}

func (g Kyverno) Enabled() bool { return true }

func (g Kyverno) GetResources() ([]pulumi.Resource, error) { return g.Resources, nil }

func (g Kyverno) GetViolations() *api.Violations {
	if g.BigBang.Configuration.Development {
		//Running in k3d requires letting certain pods run from upstream to handle load balancing
		return &api.Violations{
			RunAsRoot: []string{
				"svclb-*",
			},
			VolumeTypes: []string{
				"svclb-*",
			},
			SELinuxPolicy: []string{
				"svclb-*",
			},
			NoHostNamespace: []string{
				"svclb-*",
			},
		}
	}

	return &api.Violations{}
}

func (g Kyverno) Deploy(ctx *pulumi.Context, bb api.BigBang, deps ...pulumi.Resource) ([]pulumi.Resource, error) {
	//Create Namespace
	var namespace = defaultKyvernoNamespace
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

	ns, secret, err := DeployNamespace(ctx, namespace, bb.Configuration.ServiceMesh.Name != api.ServiceMeshNone, pc)

	//Prep the gatekeeper values based on what's been deployed

	values := make(pulumi.Map)
	/*
		{{- define "bigbang.defaults.kyverno" -}}
		replicaCount: 3

		image:
		  pullSecrets:
		  - name: private-registry
		  pullPolicy: {{ .Values.imagePullPolicy }}

		openshift: {{ .Values.openshift }}

		networkPolicies:
		  enabled: {{ .Values.networkPolicies.enabled }}
		  controlPlaneCidr: {{ .Values.networkPolicies.controlPlaneCidr }}

		serviceMonitor:
		  enabled: {{ .Values.monitoring.enabled }}
		  dashboards:
		    namespace: monitoring

		istio:
		  enabled: {{ .Values.istio.enabled }}
		{{- end -}}
	*/
	values["replicaCount"] = pulumi.Int(3)
	pullSecretMap := pulumi.MapArray{
		pulumi.Map{
			"name": pulumi.String("private-registry"),
		},
	}
	imageMap := pulumi.Map{
		"pullSecrets": pullSecretMap,
	}
	values["image"] = imageMap

	// This first set could/should be standard across apps.
	values["networkPolicies"] = pulumi.Map{
		"enabled":          pulumi.Bool(bb.Configuration.NetworkPolicies.Enabled),
		"controlPlaneCidr": pulumi.String(bb.Configuration.NetworkPolicies.ControlPlaneCIDR),
	}

	values["serviceMonitor"] = pulumi.Map{
		"enabled": pulumi.Bool(bb.Configuration.Monitoring.Name != api.MonitoringNone),
	}

	//deploy Kyverno Chart

	releaseArgs := &helmv3.ReleaseArgs{
		Chart:           pulumi.String(oci_kyverno_registry),
		Version:         pulumi.String(DEFUALT_KYVERNO_VERSION),
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(namespace),
		Name:            pulumi.String("kyverno"),
		Values:          values,
	}
	//
	release, err := helmv3.NewRelease(ctx, "kyverno", releaseArgs, pulumi.DependsOn(append(deps, ns, secret)))
	if err != nil {
		return nil, err
	}
	ctx.Export("kyverno", release)

	dockerRegistries := make([]string, 0)
	hostFilesystem := make([]string, 0)
	hostNetworkNamespcae := make([]string, 0)
	root := make([]string, 0)
	priv := make([]string, 0)
	for _, p := range bb.Packages {
		violations := p.GetViolations()

		dockerRegistries = append(dockerRegistries, violations.AllowedDockerRegistries...)
		hostFilesystem = append(hostFilesystem, violations.AllowedHostFilesystem...)
		hostNetworkNamespcae = append(hostNetworkNamespcae, violations.NoHostNamespace...)

		root = append(root, violations.RunAsRoot...)
		priv = append(priv, violations.Privileged...)
		// ctx.Log.Error(fmt.Sprintf("Number of Host Namespace Exceptions: %v", len(hostNetworkNamespcae)), nil)
	}

	if bb.Configuration.Development {
		dockerRegistries = append(dockerRegistries, "rancher/klipper-lb")
	}

	valuesPolicy := make(pulumi.Map)

	// Add IPS to config
	valuesPolicy["waitforready"] = pulumi.Map{
		"enabled":          pulumi.Bool(false),
		"imagePullSecrets": pullSecretMap,
	}
	valuesPolicy["PostInstall"] = pulumi.Map{
		"labelNamespace": pulumi.Map{
			"image": imageMap,
		},
		"probeWebhook": pulumi.Map{
			"image": imageMap,
		},
	}
	valuesPolicy["postUpgrade"] = pulumi.Map{
		"cleanupCRD": pulumi.Map{
			"image": imageMap,
		},
	}
	valuesPolicy["preUninstall"] = pulumi.Map{
		"deleteWebhookConfigurations": pulumi.Map{
			"image": imageMap,
		},
	}

	valuesPolicy["policies"] = pulumi.Map{
		"restrict-image-registries": pulumi.Map{
			"parameters": pulumi.Map{
				"allow": pulumi.ToStringArray(dockerRegistries),
			},
		},
	}

	hostNamespaceViolations := make(pulumi.MapArray, 0)
	for _, hostNamespce := range hostNetworkNamespcae {
		parts := strings.Split(hostNamespce, "/")
		if len(parts) != 2 {
			continue
		}

		hostNamespaceViolations = append(hostNamespaceViolations, pulumi.Map{
			"resources": pulumi.Map{
				"names":      pulumi.ToStringArray([]string{parts[1]}),
				"namespaces": pulumi.ToStringArray([]string{parts[0]}),
			},
		})
	}
	privViolations := make(pulumi.MapArray, 0)
	for _, p := range priv {
		parts := strings.Split(p, "/")
		if len(parts) != 2 {
			continue
		}

		privViolations = append(privViolations, pulumi.Map{
			"resources": pulumi.Map{
				"names":      pulumi.ToStringArray([]string{parts[1]}),
				"namespaces": pulumi.ToStringArray([]string{parts[0]}),
			},
		})
	}
	rootViolations := make(pulumi.MapArray, 0)
	for _, r := range root {
		parts := strings.Split(r, "/")
		if len(parts) != 2 {
			continue
		}

		rootViolations = append(rootViolations, pulumi.Map{
			"resources": pulumi.Map{
				"names":      pulumi.ToStringArray([]string{parts[1]}),
				"namespaces": pulumi.ToStringArray([]string{parts[0]}),
			},
		})
	}

	valuesPolicy["policies"].(pulumi.Map)["disallow-host-namespaces"] = pulumi.Map{
		"exclude": pulumi.Map{
			"any": hostNamespaceViolations,
		},
	}
	valuesPolicy["policies"].(pulumi.Map)["disallow-privileged-containers"] = pulumi.Map{
		"exclude": pulumi.Map{
			"any": privViolations,
		},
	}
	valuesPolicy["policies"].(pulumi.Map)["require-non-root-user"] = pulumi.Map{
		"exclude": pulumi.Map{
			"any": rootViolations,
		},
	}
	if bb.Configuration.Development {
		valuesPolicy["policies"].(pulumi.Map)["restrict-host-path-mount"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["restrict-host-path-mount-pv"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["restrict-selinux-type"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["restrict-volume-types"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["disallow-host-namespaces"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
			"exclude": pulumi.Map{
				"any": hostNamespaceViolations,
			},
		}
		valuesPolicy["policies"].(pulumi.Map)["require-non-root-user"].(pulumi.Map)["validationFailureAction"] = pulumi.String("audit")
		valuesPolicy["policies"].(pulumi.Map)["require-non-root-group"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["retrict-host-ports"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["restrict-capabilities"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["require-drop-all-capabilities"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
		valuesPolicy["policies"].(pulumi.Map)["restrict-host-ports"] = pulumi.Map{
			"validationFailureAction": pulumi.String("audit"),
		}
	}

	releasePolicyArgs := &helmv3.ReleaseArgs{
		Chart:           pulumi.String(oci_kyverno_policies_policies),
		Version:         pulumi.String(DEFAULT_KYVERNO_POLICY_VERSION),
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(namespace),
		Name:            pulumi.String("kyverno-policies"),
		Values:          valuesPolicy,
	}

	releasePolicy, err := helmv3.NewRelease(ctx, "kyverno-policies", releasePolicyArgs, pulumi.DependsOn(append(deps, release, ns, secret)))

	ctx.Export("kyverno-policy", releasePolicy)

	return []pulumi.Resource{release, releasePolicy, ns, secret}, err

}
