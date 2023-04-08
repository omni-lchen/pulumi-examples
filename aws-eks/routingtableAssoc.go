package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type RoutingtableAssoc struct {
	pulumi.ResourceState

	routingtableAssocId pulumi.IDOutput `pulumi:"routingtableAssocId"`
}

func NewRoutingTableAssoc(ctx *pulumi.Context, routingtableAssocName string, subnetId pulumi.IDOutput, routingtableId pulumi.IDOutput) (*RoutingtableAssoc, error) {
	var resource RoutingtableAssoc

	tags := NewTags(ctx)
	tags["Name"] = routingtableAssocName

	routingtableAssoc, err := ec2.NewRouteTableAssociation(ctx, routingtableAssocName, &ec2.RouteTableAssociationArgs{
		SubnetId:     subnetId,
		RouteTableId: routingtableId,
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.routingtableAssocId = routingtableAssoc.ID()
	return &resource, nil
}