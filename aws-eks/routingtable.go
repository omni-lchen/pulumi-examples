package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Routingtable struct {
	pulumi.ResourceState

	routingtableId pulumi.IDOutput `pulumi:"routingtableId"`
}

func NewRoutingTable(ctx *pulumi.Context, routingtableName string, vpcId pulumi.IDOutput, gatewayId pulumi.IDOutput) (*Routingtable, error) {
	var resource Routingtable

	tags := NewTags(ctx)
	tags["Name"] = routingtableName

	routeTable, err := ec2.NewRouteTable(ctx, routingtableName, &ec2.RouteTableArgs{
		VpcId: vpcId,
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: gatewayId,
			},
		},
		Tags: pulumi.ToStringMap(tags),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	
	resource.routingtableId = routeTable.ID()
	return &resource, nil
}