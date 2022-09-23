package flux

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/kustomize"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	v2 "github.com/runyontr/pulumi-bigbang/pkg/corev2"
)

func DeployFlux(ctx *pulumi.Context) ([]pulumi.Resource, error) {

	ks, err := kustomize.NewDirectory(ctx, "manifests",
		kustomize.DirectoryArgs{
			// Directory: pulumi.String("https://repo1.dso.mil/platform-one/big-bang/bigbang/tree/1.40.0/base/flux/"),
			// Requires this local path due to bug in talking to gitlab that prevents navigating down too
			// many groups/projects
			Directory: pulumi.String("./pkg/flux/manifests"),
		},
		// pulumi.DependsOn([]pulumi.Resource{ns, secret}),
	)
	if err != nil {
		return []pulumi.Resource{ks}, err // ignore the kustomize error that we already created the namespace
		// return err
	}

	//XXX Need to update this to pass in the Namespace object....
	creds, err := v2.DeployPullCreds(ctx, "flux-system")
	if err != nil {
		return []pulumi.Resource{ks}, err
	}

	return []pulumi.Resource{ks, creds}, err

	//Deploy the manfiests folder which was copied from upstream BB

}
