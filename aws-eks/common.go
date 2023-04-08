package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func getEnv(ctx *pulumi.Context, key string, fallback string) string {
    if value, ok := ctx.GetConfig(key); ok {
        return value
    }
    return fallback
}

func toPulumiStringArray(a []string) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, s := range a {
		res = append(res, pulumi.String(s))
	}
	return pulumi.StringArray(res)
}