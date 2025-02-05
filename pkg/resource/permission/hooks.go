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

package permission

import (
	"context"
	"strconv"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ram"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ram/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ram-controller/apis/v1alpha1"
)

const (
	StatusAttachable = "ATTACHABLE"
)

func (rm *resourceManager) customUpdatePermission(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	ko := desired.ko.DeepCopy()

	rm.setStatusDefaults(ko)

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionTrue, nil, nil)
	}

	if delta.DifferentAt("Spec.PolicyTemplate") {
		err := rm.updatePermission(ctx, desired)
		if err != nil {
			return nil, err
		}
		// resource takes time to retrieve the latest version. Syncing after
		// 30 seconds gets the job done
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	}

	return &resource{ko}, nil
}

func permissionAttachable(r *resource) bool {
	if r.ko.Status.Status == nil {
		return false
	}
	ps := *r.ko.Status.Status
	return ps == StatusAttachable
}

// In this function we decide to create a permission version
// when there's an update to the Policy statement.
// If this operation is successful, we will delete the current
// PermissionVersion, and ensure we only have one PermissionVeriosn
// at all times
//
// We decided to take this approach, because two PermissionVersions
// with the same policyTemplate are not allowed to coexist
//
// Another approach could have been retrieving all PermissionVersions
// and comparing the policyTemplates, and only updating the version.
// We did not take this approach because it would have take more
// API calls, and comparing PolicyTemplates would require more work.
func (rm *resourceManager) updatePermission(
	ctx context.Context,
	r *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.UpdatePermission")
	defer func() {
		exit(err)
	}()

	permissionArn := (*string)(r.ko.Status.ACKResourceMetadata.ARN)
	version := r.ko.Status.Version
	resp, err := rm.sdkapi.CreatePermissionVersion(
		ctx,
		&svcsdk.CreatePermissionVersionInput{
			PermissionArn:  permissionArn,
			PolicyTemplate: r.ko.Spec.PolicyTemplate,
		},
	)
	rm.metrics.RecordAPICall("UPDATE", "CreatePermissionVersion", err)
	if err != nil {
		return err
	}

	err = rm.deleteNonDefaultPermissionVersion(ctx, *permissionArn, *version)
	if err != nil {
		return err
	}

	if resp.Permission.CreationTime != nil {
		r.ko.Status.CreationTime = &metav1.Time{*resp.Permission.CreationTime}
	} else {
		r.ko.Status.CreationTime = nil
	}
	if resp.Permission.DefaultVersion != nil {
		r.ko.Status.DefaultVersion = resp.Permission.DefaultVersion
	} else {
		r.ko.Status.DefaultVersion = nil
	}
	if resp.Permission.FeatureSet != "" {
		r.ko.Status.FeatureSet = aws.String(string(resp.Permission.FeatureSet))
	} else {
		r.ko.Status.FeatureSet = nil
	}
	if resp.Permission.IsResourceTypeDefault != nil {
		r.ko.Status.IsResourceTypeDefault = resp.Permission.IsResourceTypeDefault
	} else {
		r.ko.Status.IsResourceTypeDefault = nil
	}
	if resp.Permission.LastUpdatedTime != nil {
		r.ko.Status.LastUpdatedTime = &metav1.Time{*resp.Permission.LastUpdatedTime}
	} else {
		r.ko.Status.LastUpdatedTime = nil
	}
	if resp.Permission.Name != nil {
		r.ko.Spec.Name = resp.Permission.Name
	} else {
		r.ko.Spec.Name = nil
	}
	if resp.Permission.PermissionType != "" {
		r.ko.Status.PermissionType = aws.String(string(resp.Permission.PermissionType))
	} else {
		r.ko.Status.PermissionType = nil
	}
	if resp.Permission.ResourceType != nil {
		r.ko.Spec.ResourceType = resp.Permission.ResourceType
	} else {
		r.ko.Spec.ResourceType = nil
	}
	if resp.Permission.Status != "" {
		r.ko.Status.Status = aws.String(string(resp.Permission.Status))
	} else {
		r.ko.Status.Status = nil
	}
	if resp.Permission.Version != nil {
		r.ko.Status.Version = resp.Permission.Version
	} else {
		r.ko.Status.Version = nil
	}
	if resp.Permission.Permission != nil {
		r.ko.Spec.PolicyTemplate = resp.Permission.Permission
	}

	dv, err := strconv.ParseInt(*r.ko.Status.Version, 10, 64)
	if err != nil {
		return err
	}
	newdv := int32(dv)
	_, err = rm.sdkapi.SetDefaultPermissionVersion(
		ctx,
		&svcsdk.SetDefaultPermissionVersionInput{
			PermissionArn:     permissionArn,
			PermissionVersion: &newdv,
		},
	)
	if err != nil {
		return err
	}

	return nil

}

func (rm *resourceManager) deleteNonDefaultPermissionVersion(
	ctx context.Context,
	permissionArn string,
	version string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deletePolicyVersion")
	defer func() {
		exit(err)
	}()

	v, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return err
	}
	newv := int32(v)
	_, err = rm.sdkapi.DeletePermissionVersion(
		ctx,
		&svcsdk.DeletePermissionVersionInput{
			PermissionArn:     &permissionArn,
			PermissionVersion: &newv,
		},
	)
	rm.metrics.RecordAPICall("DELETE", "DeletePolicyVersion", err)
	if err != nil {
		return err
	}
	rlog.Info(
		"deleted non-default permission version",
		"permission_arn", permissionArn,
		"permission_version", version,
	)
	return nil
}

func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {

	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func() {
		exit(err)
	}()

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
		rlog.Debug("removing tags from Permission resource", "tags", toDeleteTagKeys)
		_, err = rm.sdkapi.UntagResource(
			ctx,
			&svcsdk.UntagResourceInput{
				ResourceArn: (*string)(resourceArn),
				TagKeys:     aws.ToStringSlice(toDeleteTagKeys),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "UntagResource", err)
	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to Permission resource", "tags", toAdd)
		_, err := rm.sdkapi.TagResource(
			ctx,
			&svcsdk.TagResourceInput{
				ResourceArn: (*string)(resourceArn),
				Tags:        rm.sdkTags(toAdd),
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
