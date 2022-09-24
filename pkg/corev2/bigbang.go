package v2

import (
	"fmt"

	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func DeployBigBang(ctx *pulumi.Context, configuration api.Configuration) ([]pulumi.Resource, error) {

	/*
		For now, just gatekeeper and istio are pushed as OCI
	*/

	// Get all the packages?
	policy := GetPolicyEngine(configuration.Policy.Name)
	serviceMesh := GetServiceMesh(configuration.ServiceMesh.Name, configuration.ServiceMesh)

	bb := api.BigBang{
		Packages:      []api.BigBangPackage{policy, serviceMesh},
		Configuration: configuration,
	}
	ctx.Export("bigbang", pulumi.String(bb.Configuration.ToString()))

	resources := make([]pulumi.Resource, 0)
	for _, p := range bb.Packages {
		r, err := p.Deploy(ctx, bb, resources...)
		if err != nil {
			ctx.Log.Error(fmt.Sprintf("Error deploying package %v: %v\n", p, err), nil)
		}
		resources = append(resources, r...)
	}

	return resources, nil

}
