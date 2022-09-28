package v2

import (
	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const NEUVECTOR_VERSION = "2.2.2-bb.0"
const DEFAULT_NEUVECTOR_NAMESPACE = "neuvector"

var (
	oci_neuvector = "oci://registry.dso.mil/platform-one/big-bang/bigbang/neuvector"
)

type Neuvector struct {
	Configuration api.RuntimeSecurityConfiguration

	Resources []pulumi.Resource
}

func (n Neuvector) Enabled() bool { return true }

func (n Neuvector) GetResources() ([]pulumi.Resource, error) { return n.Resources, nil }

func (n Neuvector) NetworkPolicies() []string {
	return make([]string, 0)
}

func (n Neuvector) GetViolations() *api.Violations {
	v := api.Violations{}
	v.AllowedHostFilesystem = []string{}

	v.NoHostNamespace = []string{}

	v.Privileged = []string{"neuvector/neuvector-*"}
	v.RunAsRoot = []string{"neuvector/neuvector-*"}
	/*
			 resource DaemonSet/neuvector/neuvector-enforcer-pod was blocked due to the following policies

		    disallow-privileged-containers:
		      autogen-priviledged-containers: 'validation error: Privileged mode is not allowed.
		        The fields spec.containers[*].securityContext.privileged, spec.initContainers[*].securityContext.privileged,
		        and spec.ephemeralContainers[*].securityContext.privileged must be undefined or
		        set to false. Rule autogen-priviledged-containers failed at path /spec/template/spec/containers/0/securityContext/privileged/'
		    error: 1 error occurred:
		        * Helm release "neuvector/neuvector" was created, but failed to initialize completely. Use Helm CLI to investigate.: failed to become available within allocated timeout. Error: Helm Release neuvector/neuvector: admission webhook "validate.kyverno.svc-fail" denied the request:

		    resource DaemonSet/neuvector/neuvector-enforcer-pod was blocked due to the following policies

		    disallow-privileged-containers:
		      autogen-priviledged-containers: 'validation error: Privileged mode is not allowed.
		        The fields spec.containers[*].securityContext.privileged, spec.initContainers[*].securityContext.privileged,
		        and spec.ephemeralContainers[*].securityContext.privileged must be undefined or
		        set to false. Rule autogen-priviledged-containers failed at path /spec/template/spec/containers/0/securityContext/privileged/'

	*/

	return &v
}

func (n Neuvector) Deploy(ctx *pulumi.Context, bb api.BigBang, deps ...pulumi.Resource) ([]pulumi.Resource, error) {

	var pc PullCreds

	if len(bb.Configuration.ImagePullSecrets) > 0 {
		pc.Username = bb.Configuration.ImagePullSecrets[0].Username
		pc.Password = bb.Configuration.ImagePullSecrets[0].Password
		pc.Registry = bb.Configuration.ImagePullSecrets[0].Registry
	}

	neuvectorNamespace, ips1, err := DeployNamespace(ctx, DEFAULT_NEUVECTOR_NAMESPACE, false, pc)

	values := make(pulumi.Map)

	values["istio"] = pulumi.Map{
		"enabled": pulumi.Bool(bb.Configuration.ServiceMesh.Name == api.ServieMeshIstio),
		"mtls": pulumi.Map{
			"mode": pulumi.String("DISABLE"),
		},
		"neuvector": pulumi.Map{
			"gateways": pulumi.ToStringArray([]string{"istio-system/public"}),
		},
	}

	values["k3s"] = pulumi.Map{
		"enabled": pulumi.Bool(true),
	}
	values["manager"] = pulumi.Map{
		"env": pulumi.Map{
			"ssl": pulumi.Bool(false),
		},
	}

	releaseArgs := &helmv3.ReleaseArgs{
		Chart:   pulumi.String(oci_neuvector),
		Version: pulumi.String(NEUVECTOR_VERSION),
		// RepositoryOpts: helmv3.RepositoryOptsArgs{
		// 	Repo: pulumi.String(`oci://registry.dso.mil/platform-one/big-bang/bigbang/`),
		// },
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(DEFAULT_NEUVECTOR_NAMESPACE),
		Name:            pulumi.String("neuvector"),
		// ValueYamlFiles: pulumi.NewAssetArchive(map[string]interface{}{
		// 	"certs": pulumi.NewFileAsset("https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/ingress-certs.yaml"),
		// }),
		Values: values,
	}

	release, err := helmv3.NewRelease(ctx, "neuvector", releaseArgs, pulumi.DependsOn(append(deps, neuvectorNamespace, ips1)))
	if err != nil {
		return nil, err
	}
	ctx.Export("neuvector", release)
	return append([]pulumi.Resource{release, neuvectorNamespace, ips1}), err
}
