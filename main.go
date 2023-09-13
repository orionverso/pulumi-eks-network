package main

import (
	"k8s-network/network"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		network.NewK8sVpc(ctx, "k8s-network", &network.K8sVpcArgs{})

		return nil

	})
}
