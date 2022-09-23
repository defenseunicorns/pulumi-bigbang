// Code generated by crd2pulumi DO NOT EDIT.
// *** WARNING: Do not edit by hand unless you're certain you know what you are doing! ***

package v1alpha3

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type module struct {
	version semver.Version
}

func (m *module) Version() semver.Version {
	return m.version
}

func (m *module) Construct(ctx *pulumi.Context, name, typ, urn string) (r pulumi.Resource, err error) {
	switch typ {
	case "kubernetes:networking.istio.io/v1alpha3:VirtualService":
		r = &VirtualService{}
	default:
		return nil, fmt.Errorf("unknown resource type: %s", typ)
	}

	err = ctx.RegisterResource(typ, name, nil, r, pulumi.URN_(urn))
	return
}

func init() {
	version, err := kubernetes.PkgVersion()
	if err != nil {
		version = semver.Version{Major: 1}
	}
	pulumi.RegisterResourceModule(
		"crds",
		"networking.istio.io/v1alpha3",
		&module{version},
	)
}
