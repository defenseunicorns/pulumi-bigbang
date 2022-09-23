package v2

import (
	"fmt"
	"io/ioutil"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/runyontr/pulumi-bigbang/crds/kubernetes/networking/v1beta1"
	"github.com/runyontr/pulumi-bigbang/pkg/api"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
)

const ISTIO_VERSION = "1.14.3-bb.0"
const ISTIO_OPERATOR_VERSION = "1.14.3-bb.0"

var (
	oci_registry_operator         = "oci://registry.dso.mil/platform-one/big-bang/bigbang/istio-operator"
	oci_registry_istio            = "oci://registry.dso.mil/platform-one/big-bang/bigbang/istio"
	istio_version                 = "*"
	defaultIstioOperatorNamespace = "istio-operator"
	defaultIstioNamespace         = "istio-system"
)

type Istio struct {
	Resources []pulumi.Resource
	Domain    string

	Configuration api.ServiceMeshConfiguration
}

func (i Istio) NetworkPolicies() []string {
	return make([]string, 0)
}

func (i Istio) Enabled() bool { return true }

func (i Istio) GetResources() ([]pulumi.Resource, error) { return i.Resources, nil }

func (i Istio) GetViolations() *api.Violations {
	v := api.Violations{}
	v.AllowedHostFilesystem = []string{}

	v.NoHostNamespace = []string{}

	for _, g := range i.Configuration.Gateways {
		v.NoHostNamespace = append(v.NoHostNamespace, fmt.Sprintf("istio-system/%v", g.Name))
	}
	v.RunAsRoot = []string{}
	return &v
}

// Deploy captures how to deploy the provided Package
func (g Istio) Deploy(ctx *pulumi.Context, bb api.BigBang, deps ...pulumi.Resource) ([]pulumi.Resource, error) {
	//Create Namespace

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

	istioNamespace, ips1, err := DeployNamespace(ctx, defaultIstioNamespace, false, pc)
	istioOperatorNamespace, ips2, err := DeployNamespace(ctx, defaultIstioOperatorNamespace, false, pc)

	// Basic copying of templates/istio/operator/values.yaml
	operatorValues := make(pulumi.Map)
	operatorValues["imagePullSecrets"] = pulumi.ToStringArray([]string{"private-registry"})

	operatorValues["createNamespace"] = pulumi.Bool(false)
	operatorValues["networkPolicies"] = pulumi.Map{
		"enabled":          pulumi.Bool(bb.Configuration.NetworkPolicies.Enabled),
		"controlPlaneCidr": pulumi.String(bb.Configuration.NetworkPolicies.ControlPlaneCIDR),
	}

	//deploy Istio Operator Chart

	releaseArgs := &helmv3.ReleaseArgs{
		Chart:           pulumi.String(oci_registry_operator),
		Version:         pulumi.String(ISTIO_OPERATOR_VERSION),
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(defaultIstioOperatorNamespace),
		Name:            pulumi.String("istio-operator"),
		SkipCrds:        pulumi.Bool(false),
		SkipAwait:       pulumi.Bool(false),
		Values:          operatorValues,
	}

	release, err := helmv3.NewRelease(ctx, defaultIstioOperatorNamespace, releaseArgs, pulumi.DependsOn(append(deps, istioOperatorNamespace, ips2)))
	if err != nil {
		return nil, err
	}

	ctx.Export("istio-operator", release)

	valuesIstio := make(pulumi.Map)

	// basic copying of templates/istio/controlplane/values.yaml
	valuesIstio["monitoring"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	valuesIstio["imagePullSecrets"] = pulumi.ToStringArray([]string{"private-registry"})
	//maybe loop through packages to see if we should enable
	valuesIstio["kiali"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	valuesIstio["authservice"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	valuesIstio["domain"] = pulumi.String(bb.Configuration.ServiceMesh.Domain)

	valuesIstio["ingressGateways"] = pulumi.Map{
		"istio-ingressgateway": pulumi.Map{
			"enabled": pulumi.Bool(false),
		},
	}
	// valuesIstio["meshConfig"] = pulumi.Map{
	// 	""
	// }
	valuesIstio["gateways"] = pulumi.Map{}

	certs := make([]pulumi.Resource, 0)

	for _, gateway := range bb.Configuration.ServiceMesh.Gateways {
		//need to make a secret
		if gateway.Tls.KeyFile != "" && gateway.Tls.CertFile != "" {
			key, err := ioutil.ReadFile(gateway.Tls.KeyFile)
			if err != nil {
				ctx.Log.Error(fmt.Sprintf("Error reading Key File %v: %v", gateway.Tls.KeyFile, err), nil)
			}
			gateway.Tls.Key = string(key)
			cert, err := ioutil.ReadFile(gateway.Tls.CertFile)
			if err != nil {
				ctx.Log.Error(fmt.Sprintf("Error reading Cert File %v: %v", gateway.Tls.CertFile, err), nil)
			}
			gateway.Tls.Cert = string(cert)
		}
		certName := fmt.Sprintf("%s/%s-cert", defaultIstioNamespace, gateway.Name)

		// if gateway.Tls.Key == "" || gateway.Tls.Cert == "" {
		// 	//Its not provided in the config.  The first time through
		// 	// we can make it, but if its already in the output, we
		// 	// should load that one instead of re-deploying the cert each time
		// 	//Maybe we should see if there's already a cert in the cluster?
		// 	secret, err := corev1.GetSecret(ctx, certName,
		// 		pulumi.ID(fmt.Sprintf("%v/%v-cert")), nil)

		// 	if err != nil {
		// 		gateway.Tls.Key = secret.Data.ApplyT()
		// 		ctx.Log.Info("Secret Value not found, making cert")
		// 	}
		// 	key, cert, err := utils.CreateCerts(gateway.Domain)
		// 	if err != nil {
		// 		ctx.Log.Error(fmt.Sprintf("Error creating cert: %s", err), nil)
		// 	}
		// 	gateway.Tls.Key = key
		// 	gateway.Tls.Cert = cert
		// }

		if gateway.Tls.Key != "" && gateway.Tls.Cert != "" {
			secret, err := corev1.NewSecret(ctx, certName, &corev1.SecretArgs{
				StringData: pulumi.StringMap{
					"tls.key": pulumi.String(gateway.Tls.Key),
					"tls.crt": pulumi.String(gateway.Tls.Cert),
				},
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.Sprintf("%v-cert", gateway.Name),
					Namespace: pulumi.String("istio-system"),
				},
				Type: pulumi.String("kubernetes.io/tls"),
			})
			if err != nil {
				ctx.Log.Error(fmt.Sprintf("Error creating TLS cert for gateway %s: %s", gateway.Name, err), nil)
				return nil, err
			}
			ctx.Export(fmt.Sprintf("%v-cert", gateway.Name), secret.Metadata)
			certs = append(certs, secret)
		}

		igw := pulumi.Map{
			gateway.Name: pulumi.Map{
				"enabled": pulumi.Bool(true),
			},
		}
		gw := pulumi.Map{
			"selector": pulumi.StringMap{
				"app": pulumi.String(gateway.Name),
			},
			"autoHttpRedirect": pulumi.Map{
				"enabled": pulumi.Bool(true),
			},
			"servers": pulumi.MapArray{
				//http
				pulumi.Map{
					"hosts": pulumi.ToStringArray([]string{
						fmt.Sprintf("*.%s", gateway.Domain),
					}),
					"port": pulumi.Map{
						"name":     pulumi.String("http"),
						"number":   pulumi.Int(8080),
						"protocol": pulumi.String("HTTP"),
					},
				},
				//https
				pulumi.Map{
					"hosts": pulumi.ToStringArray([]string{
						fmt.Sprintf("*.%s", gateway.Domain),
					}),
					"port": pulumi.Map{
						"name":     pulumi.String("https"),
						"number":   pulumi.Int(8443),
						"protocol": pulumi.String("HTTPS"),
					},
					"tls": pulumi.Map{
						"credentialName": pulumi.Sprintf("%s-cert", gateway.Name),
						"mode":           pulumi.String("SIMPLE"),
					},
				},
			},
		}
		valuesIstio["ingressGateways"].(pulumi.Map)[gateway.Name] = igw
		valuesIstio["gateways"].(pulumi.Map)[gateway.Name] = gw
		valuesIstio["gateways"].(pulumi.Map)["main"] = nil
	}

	releaseIstioARgs := &helmv3.ReleaseArgs{
		Chart:   pulumi.String(oci_registry_istio),
		Version: pulumi.String(ISTIO_VERSION),
		// RepositoryOpts: helmv3.RepositoryOptsArgs{
		// 	Repo: pulumi.String(`oci://registry.dso.mil/platform-one/big-bang/bigbang/`),
		// },
		CreateNamespace: pulumi.Bool(false),
		Namespace:       pulumi.String(defaultIstioNamespace),
		Name:            pulumi.String("istio"),
		// ValueYamlFiles: pulumi.NewAssetArchive(map[string]interface{}{
		// 	"certs": pulumi.NewFileAsset("https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/ingress-certs.yaml"),
		// }),
		Values: valuesIstio,
	}

	// wait = append(wait, release)

	// // Deploy v9.6.0 version of the wordpress chart.
	releaseIstio, err := helmv3.NewRelease(ctx, "istio", releaseIstioARgs, pulumi.DependsOn(append(deps, release, istioNamespace, ips1)))
	// //  helm install bigbang --create-namespace oci://registry.dso.mil/platform-one/big-bang/bigbang/bigbang --version 1.35.0 -n bigbang -f ./chart/ingress-certs.yaml -f dev/credentials.yaml
	ctx.Export("istio", releaseIstio)
	//TODO do something better here

	return append([]pulumi.Resource{release, releaseIstio, istioNamespace, istioOperatorNamespace, ips1, ips2}, certs...), err

}

func (i Istio) SetDomain(domain string) {
	i.Domain = domain
}

func (i Istio) Ingress(ctx *pulumi.Context, ing api.Ingress) ([]pulumi.Resource, error) {
	// ctx.Export("helm-release", hr.Status.Name)
	// serviceName := (*string)(hr.Status.Name())
	// serviceName := "wordpress"
	port := ing.Port
	host := fmt.Sprintf("%s.%s", ing.Name, i.Domain)
	if ing.Hostname != "" {
		host = ing.Hostname
	}

	vs, err := v1beta1.NewVirtualService(ctx, fmt.Sprintf("istio-%s/%s", ing.Namespace, ing.Name),
		&v1beta1.VirtualServiceArgs{
			ApiVersion: pulumi.String("v1beta1"),
			Kind:       pulumi.String("VirtualService"),
			Metadata: metav1.ObjectMetaArgs{
				Namespace: pulumi.String(ing.Namespace),
				Name:      pulumi.String(ing.Name),
			},
			Spec: v1beta1.VirtualServiceSpecArgs{
				// Gateways: ,
				Gateways: pulumi.StringArray{
					pulumi.Sprintf("istio-system/%s", ing.Gateway),
					// pulumi.String("istio-system/public"),
				},
				Hosts: pulumi.StringArray{
					pulumi.Sprintf(host),
				},
				Http: v1beta1.VirtualServiceSpecHttpArray{
					v1beta1.VirtualServiceSpecHttpArgs{
						Route: v1beta1.VirtualServiceSpecHttpRouteArray{
							v1beta1.VirtualServiceSpecHttpRouteArgs{
								Destination: v1beta1.VirtualServiceSpecHttpRouteDestinationArgs{
									Host: pulumi.Sprintf("%s.%s.svc.cluster.local", ing.ServiceName, ing.Namespace),
									// Host: pulumi.StringPtr(host),
									Port: v1beta1.VirtualServiceSpecHttpRouteDestinationPortArgs{Number: port.(pulumi.IntOutput)},
								},
							},
						},
					},
				},
			},
		})
	fmt.Printf("Create VirtualService for Chart: %v\n", vs)

	//TODO Add network policies to allow ingress gateway to talk to pods being exposed

	return []pulumi.Resource{vs}, err
}
