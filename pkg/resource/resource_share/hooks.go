// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package resource_share

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ram"

	svcapitypes "github.com/aws-controllers-k8s/ram-controller/apis/v1alpha1"
)

// syncTags used to keep tags in sync by calling Create and Delete API's
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func(err error) {
		exit(err)
	}(err)

	resourceArn := latest.ko.Status.ACKResourceMetadata.ARN

	desiredTags := ToACKTags(desired.ko.Spec.Tags)
	latestTags := ToACKTags(latest.ko.Spec.Tags)

	added, _, removed := ackcompare.GetTagsDifference(latestTags, desiredTags)

	toAdd := FromACKTags(added)

	var toDeleteTagKeys []*string
	for k, _ := range removed {
		toDeleteTagKeys = append(toDeleteTagKeys, &k)
	}

	if len(toDeleteTagKeys) > 0 {
		rlog.Debug("removing tags from ResourceShare resource", "tags", toDeleteTagKeys)
		_, err = rm.sdkapi.UntagResource(
			&svcsdk.UntagResourceInput{
				ResourceShareArn: (*string)(resourceArn),
				TagKeys:          toDeleteTagKeys,
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "UntagResource", err)
		if err != nil {
			return err
		}

	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to ResourceShare resource", "tags", toAdd)
		_, err = rm.sdkapi.TagResource(
			&svcsdk.TagResourceInput{
				ResourceShareArn: (*string)(resourceArn),
				Tags:             rm.sdkTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "TagResource", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(
	tags []*svcapitypes.Tag,
) (sdktags []*svcsdk.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
}

// compareTags is a custom comparison function for comparing lists of Tag
// structs where the order of the structs in the list is not important.
func compareTags(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		desiredTags := ToACKTags(a.ko.Spec.Tags)
		latestTags := ToACKTags(b.ko.Spec.Tags)

		added, _, removed := ackcompare.GetTagsDifference(latestTags, desiredTags)

		toAdd := FromACKTags(added)
		toDelete := FromACKTags(removed)

		if len(toAdd) != 0 || len(toDelete) != 0 {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
}

func (rm *resourceManager) syncPermissions(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncPermissions")
	defer func() {
		exit(err)
	}()

	resourceArn := latest.ko.Status.ACKResourceMetadata.ARN

	desiredPermissions := desired.ko.Spec.PermissionARNs
	latestPermissions := latest.ko.Spec.PermissionARNs

	toAdd, toDelete := comparePermissionArns(desiredPermissions, latestPermissions)

	if len(toDelete) > 0 {
		rlog.Debug("disassociating permissions from ResourceShare resource", "permissionArns", toDelete)
		for _, permission := range toDelete {
			_, err = rm.sdkapi.DisassociateResourceSharePermissionWithContext(
				ctx,
				&svcsdk.DisassociateResourceSharePermissionInput{
					ResourceShareArn: (*string)(resourceArn),
					PermissionArn:    permission,
				},
			)
			rm.metrics.RecordAPICall("UPDATE", "DisassociateResourceSharePermission", err)
			if err != nil {
				return err
			}
		}
	}

	if len(toAdd) > 0 {
		rlog.Debug("associating permissions to ResourceShare resource", "permissionArns", toAdd)
		for _, permission := range toAdd {
			_, err = rm.sdkapi.AssociateResourceSharePermissionWithContext(
				ctx,
				&svcsdk.AssociateResourceSharePermissionInput{
					ResourceShareArn: (*string)(resourceArn),
					PermissionArn:    permission,
				},
			)
			rm.metrics.RecordAPICall("UPDATE", "AssociateResourceSharePermission", err)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func comparePermissionArns(a, b []*string) ([]*string, []*string) {
	toAdd := make([]*string, 0, len(a))
	toDelete := make([]*string, 0, len(a))

	am := make(map[string]bool)

	for _, v := range a {
		am[*v] = true
	}

	for _, v := range b {
		if _, ok := am[*v]; !ok {
			toDelete = append(toDelete, v)
		}
	}

	bm := make(map[string]bool)
	for _, v := range b {
		bm[*v] = true
	}

	for _, v := range a {
		if _, ok := bm[*v]; !ok {
			toAdd = append(toDelete, v)
		}
	}

	return toAdd, toDelete
}

func (rm *resourceManager) getPermissionArns(ctx context.Context, r *resource) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getPermissions")
	defer func() {
		exit(err)
	}()
	if r == nil || r.ko == nil || r.ko.Status.ACKResourceMetadata == nil || r.ko.Status.ACKResourceMetadata.ARN == nil {
		return nil
	}
	resp, err := rm.sdkapi.ListResourceSharePermissions(
		&svcsdk.ListResourceSharePermissionsInput{
			ResourceShareArn: (*string)(r.ko.Status.ACKResourceMetadata.ARN),
		},
	)
	rm.metrics.RecordAPICall("READ_MANY", "ListResourceSharePermissions", err)
	if err != nil {
		return err
	}

	if resp.Permissions != nil {
		permissionArns := make([]*string, 0, len(resp.Permissions))
		for _, p := range resp.Permissions {
			permissionArns = append(permissionArns, p.Arn)
		}
		r.ko.Spec.PermissionARNs = permissionArns
	}

	return nil
}
