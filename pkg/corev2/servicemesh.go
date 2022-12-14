package v2

import (
	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ServiceMeshInterface interface {
	api.BigBangPackage

	// pass in service name to expose and some sort of hostname?
	Ingress(*pulumi.Context, api.Ingress) ([]pulumi.Resource, error) //

	SetDomain(string)
}

func GetServiceMesh(selection api.ServiceMesh, config api.ServiceMeshConfiguration) ServiceMeshInterface {
	switch selection {
	case api.ServieMeshIstio:
		return Istio{
			Configuration: config,
		}
	case api.ServiceMeshNone:
		return nil
	}

	return Istio{}
}
