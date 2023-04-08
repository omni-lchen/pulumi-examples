package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Subnet struct {
	pulumi.ResourceState

	subnetId pulumi.IDOutput `pulumi:"subnetId"`
}

func NewSubnet(ctx *pulumi.Context, subnetName string, vpcId pulumi.IDOutput, cidr string, az string) (*Subnet, error) {
	var resource Subnet

	tags := NewTags(ctx)
	tags["Name"] = subnetName

	// Create 3 Private Subnets
	subnet, err := ec2.NewSubnet(ctx, subnetName, &ec2.SubnetArgs{
		VpcId:            vpcId,
		CidrBlock:        pulumi.String(cidr),
		AvailabilityZone: pulumi.String(az),
		Tags:             pulumi.ToStringMap(tags),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.subnetId = subnet.ID()
	return &resource, nil
}