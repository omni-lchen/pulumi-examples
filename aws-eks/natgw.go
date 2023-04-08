package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Ngw struct {
	pulumi.ResourceState

	ngwId pulumi.IDOutput `pulumi:"ngwId"`
}

// Create a NAT gateway
func NewNatGw(ctx *pulumi.Context, eipId pulumi.IDOutput, subnetId pulumi.IDOutput) (*Ngw, error) {
	var resource Ngw

	tags := NewTags(ctx)
	netGwName := NewPrefixName("natgw")
	tags["Name"] = netGwName
	natGw, err := ec2.NewNatGateway(ctx, netGwName, &ec2.NatGatewayArgs{
		// NAT Gateway with EIP
		AllocationId: eipId,
		// NAT must reside in public subnet for private instance internet access
		SubnetId: subnetId,
		Tags:     pulumi.ToStringMap(tags),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	resource.ngwId = natGw.ID()

	return &resource, nil
}