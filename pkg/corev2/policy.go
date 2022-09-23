package v2

import (
	"github.com/runyontr/pulumi-bigbang/pkg/api"
)

func GetPolicyEngine(selection api.Policy) PolicyInterface {
	switch selection {
	case api.PolicyGatekeeper:
		return Gatekeeper{}
	case api.PolicyKyverno:
		return Kyverno{}
	}

	return Gatekeeper{}
}

type PolicyInterface interface {
	api.BigBangPackage

	//not sure whta else would go in here
}
