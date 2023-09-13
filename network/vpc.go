package network

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
)

type K8sVpc struct {
	pulumi.ResourceState
	Vpc                  *ec2.Vpc
	PrivateSubnetIds     pulumi.StringArray
	PublicSubnetIds      pulumi.StringArray
	PrivateRoutetableIds pulumi.StringArray
	PublicRoutetableIds  pulumi.StringArray
	Subnets              pulumi.StringArray
}

type K8sVpcArgs struct {
}

func NewK8sVpc(ctx *pulumi.Context, name string, args *K8sVpcArgs, opts ...pulumi.ResourceOption) (*K8sVpc, error) {
	componentResource := &K8sVpc{}

	if args == nil {
		args = &K8sVpcArgs{}
	}

	// <package>:<module>:<type>
	err := ctx.RegisterComponentResource("k8s:network:K8sVpc", name, componentResource, opts...)
	if err != nil {
		return nil, err
	}

	k8svpc, err := ec2.NewVpc(ctx, "k8s-VPC", &ec2.VpcArgs{
		CidrBlock:          pulumi.StringPtr("69.0.0.0/16"),
		EnableDnsHostnames: pulumi.BoolPtr(true),
		EnableDnsSupport:   pulumi.BoolPtr(true),
	}, pulumi.Parent(componentResource))

	if err != nil {
		return nil, err
	}
	igw, err := ec2.NewInternetGateway(ctx, "k8s-InternetGateway", &ec2.InternetGatewayArgs{
		VpcId: k8svpc.ID(),
	}, pulumi.Parent(k8svpc))

	if err != nil {
		return nil, err
	}

	// _, err = ec2.NewInternetGatewayAttachment(ctx, "k8-igw-attachment", &ec2.InternetGatewayAttachmentArgs{
	// 	InternetGatewayId: igw.ID(),
	// 	VpcId:             k8svpc.ID(),
	// })
	// if err != nil {
	// 	return err
	// }

	aux := 0 // mask /20
	az := []string{"a", "b", "c"}
	privateSubnets := []*ec2.Subnet{}
	privateSubnetsIds := []pulumi.StringInput{}
	publicSubnets := []*ec2.Subnet{}
	publicSubnetsIds := []pulumi.StringInput{}
	privateRouteTableIds := []pulumi.StringInput{}
	publicRouteTableIds := []pulumi.StringInput{}

	subnetIndex := 0

	//public subnets
	for _, v := range az {
		sb, err := ec2.NewSubnet(ctx, fmt.Sprintf("k8s-Subnet-%v%s", aux, v), &ec2.SubnetArgs{
			VpcId:               k8svpc.ID(),
			AvailabilityZone:    pulumi.StringPtr(fmt.Sprintf("us-east-1%s", v)),
			CidrBlock:           pulumi.StringPtr(fmt.Sprintf("69.0.%v.0/20", aux)),
			MapPublicIpOnLaunch: pulumi.BoolPtr(true),
			Tags: pulumi.ToStringMap(map[string]string{
				"kubernetes.io/role/elb": "1", //tag for elb-controller discover subnets
			}),
		}, pulumi.Parent(k8svpc))

		if err != nil {
			return nil, err
		}

		publicSubnets = append(publicSubnets, sb)
		publicSubnetsIds = append(publicSubnetsIds, sb.ID().ToStringOutput())

		rtb, err := ec2.NewRouteTable(ctx, fmt.Sprintf("routetable-public-%v%s", subnetIndex, v), &ec2.RouteTableArgs{
			VpcId: k8svpc.ID(),
		}, pulumi.Parent(sb))

		if err != nil {
			return nil, err
		}

		publicRouteTableIds = append(publicRouteTableIds, rtb.ID())

		_, err = ec2.NewRoute(ctx, fmt.Sprintf("route-public-%v%s", subnetIndex, v), &ec2.RouteArgs{
			DestinationCidrBlock: pulumi.StringPtr("0.0.0.0/0"),
			GatewayId:            igw.ID(),
			RouteTableId:         rtb.ID(),
		}, pulumi.Parent(rtb))

		if err != nil {
			return nil, err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("rtb-asso-public-%v%s", subnetIndex, v), &ec2.RouteTableAssociationArgs{
			RouteTableId: rtb.ID(),
			SubnetId:     sb.ID(),
		}, pulumi.Parent(rtb))

		if err != nil {
			return nil, err
		}

		subnetIndex = subnetIndex + 1

		aux = aux + 16
	}

	if err != nil {
		return nil, err
	}

	ip, err := ec2.NewEip(ctx, "k8s-eip-natgateway", &ec2.EipArgs{
		Domain: pulumi.StringPtr("vpc"),
	}, pulumi.Parent(k8svpc))

	if err != nil {
		return nil, err
	}

	natgw, err := ec2.NewNatGateway(ctx, "k8s-natgateway", &ec2.NatGatewayArgs{
		AllocationId:     ip.AllocationId.ToStringPtrOutput(),
		ConnectivityType: pulumi.StringPtr("public"),
		SubnetId:         publicSubnetsIds[0], //for example
	}, pulumi.Parent(k8svpc))

	if err != nil {
		return nil, err
	}

	subnetIndex = 0
	aux = 128 //compesate
	//private subnets
	for _, v := range az {
		sb, err := ec2.NewSubnet(ctx, fmt.Sprintf("k8s-Subnet-%v%s", aux, v), &ec2.SubnetArgs{
			VpcId:            k8svpc.ID(),
			AvailabilityZone: pulumi.StringPtr(fmt.Sprintf("us-east-1%s", v)),
			CidrBlock:        pulumi.StringPtr(fmt.Sprintf("69.0.%v.0/19", aux)),
			Tags: pulumi.ToStringMap(map[string]string{
				"kubernetes.io/role/internal-elb": "1",
			}),
		}, pulumi.Parent(k8svpc))

		if err != nil {
			return nil, err
		}
		privateSubnets = append(privateSubnets, sb)
		privateSubnetsIds = append(privateSubnetsIds, sb.ID().ToStringOutput())
		aux = aux + 32 //mask 19

		rtb, err := ec2.NewRouteTable(ctx, fmt.Sprintf("routetable-private-%v%s", subnetIndex, v), &ec2.RouteTableArgs{
			VpcId: k8svpc.ID(),
		}, pulumi.Parent(sb))

		if err != nil {
			return nil, err
		}

		privateRouteTableIds = append(privateRouteTableIds, rtb.ID())

		_, err = ec2.NewRoute(ctx, fmt.Sprintf("route-private-%v%s", subnetIndex, v), &ec2.RouteArgs{
			DestinationCidrBlock: pulumi.StringPtr("0.0.0.0/0"),
			NatGatewayId:         natgw.ID().ToStringPtrOutput(),
			RouteTableId:         rtb.ID(),
		}, pulumi.Parent(rtb))

		if err != nil {
			return nil, err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("rtb-asso-private-%v%s", subnetIndex, v), &ec2.RouteTableAssociationArgs{
			RouteTableId: rtb.ID(),
			SubnetId:     sb.ID(),
		}, pulumi.Parent(rtb))

		if err != nil {
			return nil, err
		}

		subnetIndex = subnetIndex + 1

	}

	allsubnets := append(publicSubnetsIds, privateSubnetsIds...)

	if err != nil {
		return nil, err
	}

	componentResource.Vpc = k8svpc
	componentResource.PrivateSubnetIds = pulumi.StringArray(privateSubnetsIds)
	componentResource.PublicSubnetIds = pulumi.StringArray(publicSubnetsIds)
	componentResource.PrivateRoutetableIds = pulumi.StringArray(privateRouteTableIds)
	componentResource.PublicRoutetableIds = pulumi.StringArray(publicRouteTableIds)
	componentResource.Subnets = pulumi.StringArray(allsubnets)

	ctx.Export("VpcId", componentResource.Vpc.ID())
	ctx.Export("PrivateSubnetIds", componentResource.PrivateSubnetIds)
	ctx.Export("PublicSubnetIds", componentResource.PublicSubnetIds)
	ctx.Export("PrivateRoutetableIds", componentResource.PrivateRoutetableIds)
	ctx.Export("PublicRoutetableIds", componentResource.PublicRoutetableIds)
	ctx.Export("Subnets", componentResource.Subnets)

	// ctx.RegisterResourceOutputs(componentResource, pulumi.Map{})

	return componentResource, nil
}
