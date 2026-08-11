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
	"errors"
	"fmt"
	"strings"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ram"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ram/types"

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

	desiredTags, _ := convertToOrderedACKTags(desired.ko.Spec.Tags)
	latestTags, _ := convertToOrderedACKTags(latest.ko.Spec.Tags)

	added, _, removed := ackcompare.GetTagsDifference(latestTags, desiredTags)

	toAdd := fromACKTags(added, nil)

	var toDeleteTagKeys []*string
	for k, _ := range removed {
		toDeleteTagKeys = append(toDeleteTagKeys, &k)
	}

	if len(toDeleteTagKeys) > 0 {
		rlog.Debug("removing tags from ResourceShare resource", "tags", toDeleteTagKeys)
		_, err = rm.sdkapi.UntagResource(
			ctx,
			&svcsdk.UntagResourceInput{
				ResourceShareArn: (*string)(resourceArn),
				TagKeys:          aws.ToStringSlice(toDeleteTagKeys),
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
			ctx,
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
) (sdktags []svcsdktypes.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
}

// customPreCompare compares the fields excluded via `compare.is_ignored`.
func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	compareTags(delta, a, b)
	comparePermissionARNs(delta, a, b)
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
		desiredTags, _ := convertToOrderedACKTags(a.ko.Spec.Tags)
		latestTags, _ := convertToOrderedACKTags(b.ko.Spec.Tags)

		added, _, removed := ackcompare.GetTagsDifference(latestTags, desiredTags)

		toAdd := fromACKTags(added, nil)
		toDelete := fromACKTags(removed, nil)

		if len(toAdd) != 0 || len(toDelete) != 0 {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
}

// e.g. arn:aws:ram::aws:permission/AWSRAMDefaultPermissionPrefixList
const awsDefaultPermissionARNFragment = ":ram::aws:permission/AWSRAMDefaultPermission"

func isDefaultPermissionARN(arn string) bool {
	return strings.Contains(arn, awsDefaultPermissionARNFragment)
}

// excludeUndeclaredDefaultPermissions drops the default permissions RAM
// attached on its own, which are AWS-owned state rather than drift.
func excludeUndeclaredDefaultPermissions(
	declared []*string,
	observed []*string,
) []*string {
	declaredARNs := make(map[string]struct{}, len(declared))
	for _, arn := range declared {
		if arn != nil {
			declaredARNs[*arn] = struct{}{}
		}
	}

	kept := make([]*string, 0, len(observed))
	for _, arn := range observed {
		if arn == nil {
			continue
		}
		if _, ok := declaredARNs[*arn]; !ok && isDefaultPermissionARN(*arn) {
			continue
		}
		kept = append(kept, arn)
	}

	return kept
}

func comparePermissionARNs(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	desired := aws.ToStringSlice(a.ko.Spec.PermissionARNs)
	latest := aws.ToStringSlice(
		excludeUndeclaredDefaultPermissions(a.ko.Spec.PermissionARNs, b.ko.Spec.PermissionARNs),
	)

	if !ackcompare.SliceStringEqual(desired, latest) {
		delta.Add("Spec.PermissionARNs", a.ko.Spec.PermissionARNs, b.ko.Spec.PermissionARNs)
	}
}

type resolvedPermission struct {
	arn          string
	resourceType string
}

func checkDuplicatePermissionTypes(resolved []resolvedPermission) error {
	byType := make(map[string]string, len(resolved))
	for _, r := range resolved {
		if existing, ok := byType[r.resourceType]; ok && existing != r.arn {
			return ackerr.NewTerminalError(fmt.Errorf(
				"spec.permissionARNs declares more than one permission for resource type %s: %s and %s. RAM allows one permission per resource type",
				r.resourceType, existing, r.arn,
			))
		}
		byType[r.resourceType] = r.arn
	}
	return nil
}

func (rm *resourceManager) validateDeclaredPermissions(
	ctx context.Context,
	declared []*string,
) error {
	resolved := make([]resolvedPermission, 0, len(declared))

	for _, arn := range declared {
		if arn == nil {
			continue
		}
		resp, err := rm.sdkapi.GetPermission(
			ctx,
			&svcsdk.GetPermissionInput{PermissionArn: arn},
		)
		rm.metrics.RecordAPICall("READ_ONE", "GetPermission", err)
		if err != nil {
			var notFound *svcsdktypes.UnknownResourceException
			if errors.As(err, &notFound) {
				continue
			}
			return err
		}
		if resp.Permission == nil || resp.Permission.ResourceType == nil {
			continue
		}
		resolved = append(resolved, resolvedPermission{
			arn:          *arn,
			resourceType: *resp.Permission.ResourceType,
		})
	}

	return checkDuplicatePermissionTypes(resolved)
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
	if err = rm.validateDeclaredPermissions(ctx, desiredPermissions); err != nil {
		return err
	}

	latestPermissions := excludeUndeclaredDefaultPermissions(desiredPermissions, latest.ko.Spec.PermissionARNs)

	toAdd, toDelete := compareStringSlices(desiredPermissions, latestPermissions)

	if len(toDelete) > 0 {
		rlog.Debug("disassociating permissions from ResourceShare resource", "permissionArns", toDelete)
		for _, permission := range toDelete {
			_, err = rm.sdkapi.DisassociateResourceSharePermission(
				ctx,
				&svcsdk.DisassociateResourceSharePermissionInput{
					ResourceShareArn: (*string)(resourceArn),
					PermissionArn:    &permission,
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
			_, err = rm.sdkapi.AssociateResourceSharePermission(
				ctx,
				&svcsdk.AssociateResourceSharePermissionInput{
					ResourceShareArn: (*string)(resourceArn),
					PermissionArn:    &permission,
					Replace:          aws.Bool(true),
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

func compareStringSlices(a, b []*string) ([]string, []string) {
	toAdd := make([]string, 0, len(a))
	toDelete := make([]string, 0, len(a))

	am := make(map[string]bool)

	for _, v := range a {
		am[*v] = true
	}

	for _, v := range b {
		if _, ok := am[*v]; !ok {
			toDelete = append(toDelete, *v)
		}
	}

	bm := make(map[string]bool)
	for _, v := range b {
		bm[*v] = true
	}

	for _, v := range a {
		if _, ok := bm[*v]; !ok {
			toAdd = append(toAdd, *v)
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
		ctx,
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

func (rm *resourceManager) syncResourceShareResources(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncResources")
	defer func() {
		exit(err)
	}()

	resourceShareArn := latest.ko.Status.ACKResourceMetadata.ARN

	desiredPrincipals := desired.ko.Spec.Principals
	latestPrincipals := latest.ko.Spec.Principals

	desiredResourceArns := desired.ko.Spec.ResourceARNs
	latestResourceArns := latest.ko.Spec.ResourceARNs

	desiredSources := desired.ko.Spec.Sources
	latestSources := latest.ko.Spec.Sources

	toAddPrincipals, toDeletePrincipals := compareStringSlices(desiredPrincipals, latestPrincipals)
	toAddResources, toDeleteResources := compareStringSlices(desiredResourceArns, latestResourceArns)
	toAddSources, toDeleteSources := compareStringSlices(desiredSources, latestSources)

	if len(toDeletePrincipals)+len(toDeleteResources)+len(toDeleteSources) > 0 {
		rlog.Debug("disassociationg resources from ResourceShare")
		_, err = rm.sdkapi.DisassociateResourceShare(
			ctx,
			&svcsdk.DisassociateResourceShareInput{
				ResourceShareArn: (*string)(resourceShareArn),
				Principals:       toDeletePrincipals,
				ResourceArns:     toDeleteResources,
				Sources:          toDeleteSources,
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "DisassociateResourceShare", err)
		if err != nil {
			return err
		}
	}

	if len(toAddPrincipals)+len(toAddResources)+len(toAddSources) > 0 {
		rlog.Debug("associating resources to ResourceShare")
		_, err = rm.sdkapi.AssociateResourceShare(
			ctx,
			&svcsdk.AssociateResourceShareInput{
				ResourceShareArn: (*string)(resourceShareArn),
				Principals:       toAddPrincipals,
				ResourceArns:     toAddResources,
				Sources:          toAddSources,
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "AssociateResourceShare", err)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rm *resourceManager) getResourceShareAssociations(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getResourceShareAssociations")
	defer func() {
		exit(err)
	}()
	if r == nil || r.ko == nil || r.ko.Status.ACKResourceMetadata == nil || r.ko.Status.ACKResourceMetadata.ARN == nil {
		return nil
	}
	resourceArn := r.ko.Status.ACKResourceMetadata.ARN
	r.ko.Spec.Principals, err = rm.setResourceShareAssociation(ctx, svcsdktypes.ResourceShareAssociationTypePrincipal, *((*string)(resourceArn)))
	if err != nil {
		return err
	}
	r.ko.Spec.ResourceARNs, err = rm.setResourceShareAssociation(ctx, svcsdktypes.ResourceShareAssociationTypeResource, *((*string)(resourceArn)))
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) setResourceShareAssociation(
	ctx context.Context,
	resresourceType svcsdktypes.ResourceShareAssociationType,
	resourceArn string,
) (slices []*string, err error) {

	resp, err := rm.sdkapi.GetResourceShareAssociations(
		ctx,
		&svcsdk.GetResourceShareAssociationsInput{
			AssociationType:   resresourceType,
			ResourceShareArns: []string{resourceArn},
		},
	)

	slices = make([]*string, 0)
	rm.metrics.RecordAPICall("READ_MANY", "GetResourceShareAssociations", err)
	if err != nil {
		return nil, err
	}
	if resp.ResourceShareAssociations != nil {
		for _, p := range resp.ResourceShareAssociations {
			if p.Status == svcsdktypes.ResourceShareAssociationStatusAssociated {
				slices = append(slices, p.AssociatedEntity)
			}
		}
	}
	return slices, err
}

func (rm *resourceManager) newTag(
	c svcapitypes.Tag,
) svcsdktypes.Tag {
	res := svcsdktypes.Tag{}
	if c.Key != nil {
		res.Key = c.Key
	}
	if c.Value != nil {
		res.Value = c.Value
	}

	return res
}
