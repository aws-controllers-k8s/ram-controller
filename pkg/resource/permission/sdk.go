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

// Code generated by ack-generate. DO NOT EDIT.

package permission

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ram"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ram/types"
	smithy "github.com/aws/smithy-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ram-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = strings.ToLower("")
	_ = &svcsdk.Client{}
	_ = &svcapitypes.Permission{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
	_ = fmt.Sprintf("")
	_ = &ackrequeue.NoRequeue{}
	_ = &aws.Config{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()
	// If any required fields in the input shape are missing, AWS resource is
	// not created yet. Return NotFound here to indicate to callers that the
	// resource isn't yet created.
	if rm.requiredFieldsMissingFromReadOneInput(r) {
		return nil, ackerr.NotFound
	}

	input, err := rm.newDescribeRequestPayload(r)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.GetPermissionOutput
	resp, err = rm.sdkapi.GetPermission(ctx, input)
	rm.metrics.RecordAPICall("READ_ONE", "GetPermission", err)
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) && awsErr.ErrorCode() == "UnknownResourceException" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.Permission.Arn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.Permission.Arn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.Permission.CreationTime != nil {
		ko.Status.CreationTime = &metav1.Time{*resp.Permission.CreationTime}
	} else {
		ko.Status.CreationTime = nil
	}
	if resp.Permission.DefaultVersion != nil {
		ko.Status.DefaultVersion = resp.Permission.DefaultVersion
	} else {
		ko.Status.DefaultVersion = nil
	}
	if resp.Permission.FeatureSet != "" {
		ko.Status.FeatureSet = aws.String(string(resp.Permission.FeatureSet))
	} else {
		ko.Status.FeatureSet = nil
	}
	if resp.Permission.IsResourceTypeDefault != nil {
		ko.Status.IsResourceTypeDefault = resp.Permission.IsResourceTypeDefault
	} else {
		ko.Status.IsResourceTypeDefault = nil
	}
	if resp.Permission.LastUpdatedTime != nil {
		ko.Status.LastUpdatedTime = &metav1.Time{*resp.Permission.LastUpdatedTime}
	} else {
		ko.Status.LastUpdatedTime = nil
	}
	if resp.Permission.Name != nil {
		ko.Spec.Name = resp.Permission.Name
	} else {
		ko.Spec.Name = nil
	}
	if resp.Permission.PermissionType != "" {
		ko.Status.PermissionType = aws.String(string(resp.Permission.PermissionType))
	} else {
		ko.Status.PermissionType = nil
	}
	if resp.Permission.ResourceType != nil {
		ko.Spec.ResourceType = resp.Permission.ResourceType
	} else {
		ko.Spec.ResourceType = nil
	}
	if resp.Permission.Status != "" {
		ko.Status.Status = aws.String(string(resp.Permission.Status))
	} else {
		ko.Status.Status = nil
	}
	if resp.Permission.Tags != nil {
		f11 := []*svcapitypes.Tag{}
		for _, f11iter := range resp.Permission.Tags {
			f11elem := &svcapitypes.Tag{}
			if f11iter.Key != nil {
				f11elem.Key = f11iter.Key
			}
			if f11iter.Value != nil {
				f11elem.Value = f11iter.Value
			}
			f11 = append(f11, f11elem)
		}
		ko.Spec.Tags = f11
	} else {
		ko.Spec.Tags = nil
	}
	if resp.Permission.Version != nil {
		ko.Status.Version = resp.Permission.Version
	} else {
		ko.Status.Version = nil
	}

	rm.setStatusDefaults(ko)
	if resp.Permission.Permission != nil {
		ko.Spec.PolicyTemplate = resp.Permission.Permission
	}

	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadOneInput returns true if there are any fields
// for the ReadOne Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadOneInput(
	r *resource,
) bool {
	return (r.ko.Status.ACKResourceMetadata == nil || r.ko.Status.ACKResourceMetadata.ARN == nil)

}

// newDescribeRequestPayload returns SDK-specific struct for the HTTP request
// payload of the Describe API call for the resource
func (rm *resourceManager) newDescribeRequestPayload(
	r *resource,
) (*svcsdk.GetPermissionInput, error) {
	res := &svcsdk.GetPermissionInput{}

	if r.ko.Status.ACKResourceMetadata != nil && r.ko.Status.ACKResourceMetadata.ARN != nil {
		res.PermissionArn = (*string)(r.ko.Status.ACKResourceMetadata.ARN)
	}

	return res, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API and
// returns a copy of the resource with resource fields (in both Spec and
// Status) filled in with values from the CREATE API operation's Output shape.
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer func() {
		exit(err)
	}()
	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.CreatePermissionOutput
	_ = resp
	resp, err = rm.sdkapi.CreatePermission(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreatePermission", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.Permission.Arn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.Permission.Arn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.Permission.CreationTime != nil {
		ko.Status.CreationTime = &metav1.Time{*resp.Permission.CreationTime}
	} else {
		ko.Status.CreationTime = nil
	}
	if resp.Permission.DefaultVersion != nil {
		ko.Status.DefaultVersion = resp.Permission.DefaultVersion
	} else {
		ko.Status.DefaultVersion = nil
	}
	if resp.Permission.FeatureSet != "" {
		ko.Status.FeatureSet = aws.String(string(resp.Permission.FeatureSet))
	} else {
		ko.Status.FeatureSet = nil
	}
	if resp.Permission.IsResourceTypeDefault != nil {
		ko.Status.IsResourceTypeDefault = resp.Permission.IsResourceTypeDefault
	} else {
		ko.Status.IsResourceTypeDefault = nil
	}
	if resp.Permission.LastUpdatedTime != nil {
		ko.Status.LastUpdatedTime = &metav1.Time{*resp.Permission.LastUpdatedTime}
	} else {
		ko.Status.LastUpdatedTime = nil
	}
	if resp.Permission.Name != nil {
		ko.Spec.Name = resp.Permission.Name
	} else {
		ko.Spec.Name = nil
	}
	if resp.Permission.PermissionType != "" {
		ko.Status.PermissionType = aws.String(string(resp.Permission.PermissionType))
	} else {
		ko.Status.PermissionType = nil
	}
	if resp.Permission.ResourceType != nil {
		ko.Spec.ResourceType = resp.Permission.ResourceType
	} else {
		ko.Spec.ResourceType = nil
	}
	if resp.Permission.Status != nil {
		ko.Status.Status = resp.Permission.Status
	} else {
		ko.Status.Status = nil
	}
	if resp.Permission.Tags != nil {
		f10 := []*svcapitypes.Tag{}
		for _, f10iter := range resp.Permission.Tags {
			f10elem := &svcapitypes.Tag{}
			if f10iter.Key != nil {
				f10elem.Key = f10iter.Key
			}
			if f10iter.Value != nil {
				f10elem.Value = f10iter.Value
			}
			f10 = append(f10, f10elem)
		}
		ko.Spec.Tags = f10
	} else {
		ko.Spec.Tags = nil
	}
	if resp.Permission.Version != nil {
		ko.Status.Version = resp.Permission.Version
	} else {
		ko.Status.Version = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreatePermissionInput, error) {
	res := &svcsdk.CreatePermissionInput{}

	if r.ko.Spec.Name != nil {
		res.Name = r.ko.Spec.Name
	}
	if r.ko.Spec.PolicyTemplate != nil {
		res.PolicyTemplate = r.ko.Spec.PolicyTemplate
	}
	if r.ko.Spec.ResourceType != nil {
		res.ResourceType = r.ko.Spec.ResourceType
	}
	if r.ko.Spec.Tags != nil {
		f3 := []svcsdktypes.Tag{}
		for _, f3iter := range r.ko.Spec.Tags {
			f3elem := &svcsdktypes.Tag{}
			if f3iter.Key != nil {
				f3elem.Key = f3iter.Key
			}
			if f3iter.Value != nil {
				f3elem.Value = f3iter.Value
			}
			f3 = append(f3, *f3elem)
		}
		res.Tags = f3
	}

	return res, nil
}

// sdkUpdate patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	return rm.customUpdatePermission(ctx, desired, latest, delta)
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeletePermissionOutput
	_ = resp
	resp, err = rm.sdkapi.DeletePermission(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeletePermission", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeletePermissionInput, error) {
	res := &svcsdk.DeletePermissionInput{}

	if r.ko.Status.ACKResourceMetadata != nil && r.ko.Status.ACKResourceMetadata.ARN != nil {
		res.PermissionArn = (*string)(r.ko.Status.ACKResourceMetadata.ARN)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.Permission,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	if err == nil {
		return false
	}

	var terminalErr smithy.APIError
	if !errors.As(err, &terminalErr) {
		return false
	}
	switch terminalErr.ErrorCode() {
	case "InvalidParameterException":
		return true
	default:
		return false
	}
}
