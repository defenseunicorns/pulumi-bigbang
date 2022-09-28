package v2

import (
	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
)

type RuntimeSecurityInterface interface {
	api.BigBangPackage
}

func GetRuntimeSecurity(selection api.RuntimeSecurity, config api.RuntimeSecurityConfiguration) RuntimeSecurityInterface {
	switch selection {
	case api.RuntimeSecurityNeuvector:
		return Neuvector{
			Configuration: config,
		}
	case api.RuntimeSecurityPrismaCloud:
		return PrismaCloudCompute{
			Configuration: config,
		}
	case api.RuntimeSecurityNone:
		return nil
	}

	return Neuvector{}
}
