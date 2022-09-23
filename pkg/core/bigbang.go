package core

import (
	"io/ioutil"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func DeployBigBang(ctx *pulumi.Context, deps []pulumi.Resource) (pulumi.Resource, error) {

	c := config.New(ctx, "")
	username := config.Get(ctx, "registry.username")
	password := c.Require("registry.password")
	registry := c.Get("registry.registry")

	publicCertKey, err := ioutil.ReadFile(c.Get("keyFile"))
	if err != nil {
		return nil, err
	}

	publicCert, err := ioutil.ReadFile(c.Get("certFile"))
	if err != nil {
		return nil, err
	}

	values := make(pulumi.Map)
	values["registryCredentials"] = pulumi.StringMap{
		"registry": pulumi.String(registry),
		"username": pulumi.String(username),
		"password": pulumi.String(password),
	}
	values["domain"] = pulumi.String("bigbang.dev")

	values["istio"] = pulumi.Map{
		"enabled": pulumi.Bool(true),
		"gateways": pulumi.Map{
			"public": pulumi.Map{
				"tls": pulumi.Map{
					"key":  pulumi.String(string(publicCertKey)),
					"cert": pulumi.String(string(publicCert)),
				},
			},
		},
	}

	values["kyverno"] = pulumi.Map{
		"enabled": pulumi.Bool(true),
	}

	values["fluentbit"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	values["eckoperator"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	values["logging"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	values["loki"] = pulumi.Map{
		"enabled": pulumi.Bool(true),
	}
	values["promtail"] = pulumi.Map{
		"enabled": pulumi.Bool(true),
	}
	values["jaeger"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}
	values["tempo"] = pulumi.Map{
		"enabled": pulumi.Bool(true),
	}
	values["clusterAuditor"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
	}

	values["gatekeeper"] = pulumi.Map{
		"enabled": pulumi.Bool(false),
		"values": pulumi.Map{
			"violations": pulumi.Map{
				"allowedDockerRegistries": pulumi.Map{
					// "enabled": pulumi.Bool(false),
					"parameters": pulumi.Map{
						"repos": pulumi.StringArray{pulumi.String("rancher/klipper-lb")},
					},
				},
				"hostNetworking": pulumi.Map{
					"match": pulumi.Map{
						"excludedNamespaces": pulumi.StringArray{pulumi.String("istio-system")},
					},
				},
			},
		},
	}

	/// deps
	// foo := make([]pulumi.AnyOutput,0)
	// for k,v := range deps {
	// 	foo = append(foo, )
	// }

	release := &helmv3.ReleaseArgs{
		Chart:   pulumi.String("oci://registry.dso.mil/platform-one/big-bang/bigbang/bigbang"),
		Version: pulumi.String("1.40.0"),
		// RepositoryOpts: helmv3.RepositoryOptsArgs{
		// 	Repo: pulumi.String(`oci://registry.dso.mil/platform-one/big-bang/bigbang/`),
		// },
		CreateNamespace: pulumi.Bool(true),
		Namespace:       pulumi.String("bigbang"),

		// ValueYamlFiles: pulumi.NewAssetArchive(map[string]interface{}{
		// 	"certs": pulumi.NewFileAsset("https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/ingress-certs.yaml"),
		// }),
		Values: values,
	}
	// Deploy v9.6.0 version of the wordpress chart.
	bb, err := helmv3.NewRelease(ctx, "bigbang", release, pulumi.DependsOn(deps))
	//  helm install bigbang --create-namespace oci://registry.dso.mil/platform-one/big-bang/bigbang/bigbang --version 1.35.0 -n bigbang -f ./chart/ingress-certs.yaml -f dev/credentials.yaml
	if err != nil {
		return bb, err
	}

	ctx.Export("domain", values["domain"])
	ctx.Export("istio.enabled", values["istio"].(pulumi.Map)["enabled"])
	ctx.Export("registry.username", values["registryCredentials"].(pulumi.StringMap)["username"])
	ctx.Export("registry.registry", values["registryCredentials"].(pulumi.StringMap)["registry"])
	ctx.Export("registry.password", pulumi.ToSecret(values["registryCredentials"].(pulumi.StringMap)["password"]))
	ctx.Export("istio.tenantGateway", pulumi.String("public"))

	//export some more istio things

	// pulumi.AdditionalSecretOutputs([]string{"registry.password"})

	return bb, nil
}
