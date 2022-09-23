package v2

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/runyontr/pulumi-bigbang/pkg/api"
	"github.com/runyontr/pulumi-bigbang/pkg/k8s"
)

type Chart struct {
	Chart     string
	Version   string
	Name      string
	Namespace string
	ValueFile string
	Values    map[string]interface{}
	Repo      string
}

func DeployChart(ctx *pulumi.Context, chart Chart, bb *api.BigBang, deps ...pulumi.Resource) (*helm.Release, error) {
	var fileAsset pulumi.Asset

	if chart.ValueFile != "" {
		fileAsset = pulumi.NewFileAsset(chart.ValueFile)
	}

	hr, err := helm.NewRelease(ctx, chart.Name, &helm.ReleaseArgs{
		Chart:     pulumi.String(chart.Name),
		Version:   pulumi.String(chart.Version),
		Namespace: pulumi.String(chart.Namespace),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String(chart.Repo),
		},
		ValueYamlFiles: pulumi.AssetOrArchiveArray{fileAsset},
		// Postrender: "",
		// ValueYamlFiles: []pulumi.Asset{}.(pulumi.AssetOrArchiveArray),
	}, pulumi.DependsOn(deps))

	//lets see if there's a service that has the app.kubernetes.io/name label with value chart.Name.  Assume that's what we should expose
	// For now, since we know its wordpress, we can just use the release name which seems to be that object

	serviceMesh := GetServiceMesh(bb.Configuration.ServiceMesh.Name, bb.Configuration.ServiceMesh)
	domain := ""
	for _, gw := range bb.Configuration.ServiceMesh.Gateways {
		if gw.Name == "public" {
			domain = gw.Domain
		}
	}
	if domain == "" {
		domain = bb.Configuration.ServiceMesh.Domain
	}
	serviceMesh.SetDomain(domain)

	// host := hr.Status.Name().ApplyT(func(name string) string {
	// 	return fmt.Sprintf("%s.%s.svc.cluster.local", name, chart.Namespace)
	// }).(pulumi.StringOutput)

	//something fake to wait
	svc := pulumi.All(hr.Status.Namespace(), hr.Status.Name()).
		ApplyT(func(r interface{}) (string, error) {
			arr := r.([]interface{})
			namespace := arr[0].(*string)
			name := arr[1].(*string)

			serviceName, _, err := k8s.GetServiceName(*namespace, "app.kubernetes.io/instance", *name) //wordpress has this
			if err != nil {
				return "", err
			}
			if serviceName != "" {
				return serviceName, nil
			}
			serviceName, _, err = k8s.GetServiceName(*namespace, "app.kubernetes.io/name", *name) //podinfo has this

			return serviceName, nil

		}).(pulumi.StringOutput)

	svcName := pulumi.Sprintf("%v", svc)

	port := svcName.ApplyT(func(name string) int {
		port, err := k8s.GetServicePort(chart.Namespace, name)
		if err != nil {
			return -1
		}
		return port
	}).(pulumi.IntOutput)

	//can i find the right service?
	// using the kubernetes go client
	if err != nil {
		ctx.Log.Error(fmt.Sprintf("Error getting service name: %v", err), nil)
	}

	_, err = serviceMesh.Ingress(ctx, api.Ingress{
		Name:        chart.Name,
		Namespace:   chart.Namespace,
		Port:        port,
		Gateway:     "public", //could look up things from here eventually
		ServiceName: svcName,
		Hostname:    fmt.Sprintf("%s.%s", chart.Name, domain),
	})
	if err != nil {
		ctx.Log.Error(fmt.Sprintf("Error adding Ingress: %v", err), nil)
	}

	return hr, err
}
