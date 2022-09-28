package v2

import (
	"fmt"

	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const PRISMA_VERSION = "0.10.0-bb.2"
const DEFAULT_PRISMA_NAMESPACE = "prisma"

var (
	oci_prisma = "oci://registry.dso.mil/platform-one/big-bang/bigbang/twistlock"
)

type PrismaCloudCompute struct {
	Configuration api.RuntimeSecurityConfiguration

	Resources []pulumi.Resource
}

func (pcc PrismaCloudCompute) Enabled() bool { return true }

func (pcc PrismaCloudCompute) GetResources() ([]pulumi.Resource, error) { return pcc.Resources, nil }

func (pcc PrismaCloudCompute) NetworkPolicies() []string {
	return make([]string, 0)
}

func (pcc PrismaCloudCompute) GetViolations() *api.Violations {
	v := api.Violations{}
	v.AllowedHostFilesystem = []string{}

	v.NoHostNamespace = []string{}

	v.Privileged = []string{"prisma/twistlock-defender-ds*"}
	v.RunAsRoot = []string{"prisma/twistlock-defender-ds*"}
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

func (pcc PrismaCloudCompute) Deploy(ctx *pulumi.Context, bb api.BigBang, deps ...pulumi.Resource) ([]pulumi.Resource, error) {

	var pc PullCreds

	if len(bb.Configuration.ImagePullSecrets) > 0 {
		pc.Username = bb.Configuration.ImagePullSecrets[0].Username
		pc.Password = bb.Configuration.ImagePullSecrets[0].Password
		pc.Registry = bb.Configuration.ImagePullSecrets[0].Registry
	}

	namespace, ips1, err := DeployNamespace(ctx, DEFAULT_PRISMA_NAMESPACE, true, pc)

	values := make(pulumi.Map)

	values["imagePullSecrets"] = pulumi.MapArray{
		pulumi.Map{
			"name": pulumi.String("private-registry"),
		},
	}
	// This first set could/should be standard across apps.
	values["networkPolicies"] = pulumi.Map{
		"enabled":          pulumi.Bool(bb.Configuration.NetworkPolicies.Enabled),
		"controlPlaneCidr": pulumi.String(bb.Configuration.NetworkPolicies.ControlPlaneCIDR),
	}

	domain := bb.Configuration.ServiceMesh.Domain // default
	gateway := "public"
	if domain == "" && len(bb.Configuration.ServiceMesh.Gateways) > 0 {
		domain = bb.Configuration.ServiceMesh.Gateways[0].Domain
		gateway = bb.Configuration.ServiceMesh.Gateways[0].Name
	}
	if domain == "" {
		domain = "bigbang.dev"
	}

	values["istio"] = pulumi.Map{
		"enabled": pulumi.Bool(bb.Configuration.ServiceMesh.Name == api.ServieMeshIstio),
		"mtls": pulumi.Map{
			"mode": pulumi.String("DISABLE"),
		},
		"console": pulumi.Map{
			"gateways": pulumi.ToStringArray([]string{fmt.Sprintf("istio-system/%v", gateway)}), //could look this up  on the BB config
		},
	}
	values["domain"] = pulumi.String(domain)

	releaseArgs := &helmv3.ReleaseArgs{
		Chart:   pulumi.String(oci_prisma),
		Version: pulumi.String(PRISMA_VERSION),
		// RepositoryOpts: helmv3.RepositoryOptsArgs{
		// 	Repo: pulumi.String(`oci://registry.dso.mil/platform-one/big-bang/bigbang/`),
		// },
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(DEFAULT_PRISMA_NAMESPACE),
		Name:            pulumi.String("prisma"),
		// ValueYamlFiles: pulumi.NewAssetArchive(map[string]interface{}{
		// 	"certs": pulumi.NewFileAsset("https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/ingress-certs.yaml"),
		// }),
		Values: values,
	}

	release, err := helmv3.NewRelease(ctx, "prisma", releaseArgs, pulumi.DependsOn(append(deps, namespace, ips1)))
	if err != nil {
		return nil, err
	}
	ctx.Export("prisma", release)
	return append([]pulumi.Resource{release, namespace, ips1}), err
}
