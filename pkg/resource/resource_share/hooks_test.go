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
	"errors"
	"sort"
	"strings"
	"testing"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	"github.com/aws/aws-sdk-go-v2/aws"

	svcapitypes "github.com/aws-controllers-k8s/ram-controller/apis/v1alpha1"
)

const (
	defaultPrefixListPermission = "arn:aws:ram::aws:permission/AWSRAMDefaultPermissionPrefixList"
	defaultSubnetPermission     = "arn:aws:ram::aws:permission/AWSRAMDefaultPermissionSubnet"
	customSubnetPermission      = "arn:aws:ram:us-west-2:123456789012:permission/my-subnet-permission"
)

func newResourceShareWithPermissions(arns []string) *resource {
	ko := &svcapitypes.ResourceShare{}
	if arns != nil {
		ko.Spec.PermissionARNs = aws.StringSlice(arns)
	}
	return &resource{ko}
}

func TestComparePermissionARNs(t *testing.T) {
	tests := []struct {
		name      string
		desired   []string
		latest    []string
		different bool
	}{
		{
			name:      "undeclared default permission is not drift",
			desired:   nil,
			latest:    []string{defaultPrefixListPermission},
			different: false,
		},
		{
			name:      "undeclared default permissions alongside a declared one are not drift",
			desired:   []string{customSubnetPermission},
			latest:    []string{customSubnetPermission, defaultPrefixListPermission},
			different: false,
		},
		{
			name:      "explicitly declared default permission is not drift",
			desired:   []string{defaultPrefixListPermission},
			latest:    []string{defaultPrefixListPermission},
			different: false,
		},
		{
			name:      "order of declared permissions is not drift",
			desired:   []string{customSubnetPermission, defaultPrefixListPermission},
			latest:    []string{defaultPrefixListPermission, customSubnetPermission},
			different: false,
		},
		{
			name:      "undeclared non-default permission is drift",
			desired:   nil,
			latest:    []string{customSubnetPermission},
			different: true,
		},
		{
			name:      "declared permission missing from the share is drift",
			desired:   []string{customSubnetPermission},
			latest:    []string{defaultPrefixListPermission},
			different: true,
		},
		{
			name:      "declared permission not yet applied over the default for its own type is drift",
			desired:   []string{customSubnetPermission},
			latest:    []string{defaultSubnetPermission},
			different: true,
		},
		{
			name:      "declared default permission missing from the share is drift",
			desired:   []string{defaultSubnetPermission},
			latest:    nil,
			different: true,
		},
		{
			name:      "customer managed permission is never treated as a default",
			desired:   nil,
			latest:    []string{"arn:aws:ram:us-west-2:123456789012:permission/AWSRAMDefaultPermissionSubnet"},
			different: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := newResourceShareWithPermissions(test.desired)
			b := newResourceShareWithPermissions(test.latest)

			delta := ackcompare.NewDelta()
			comparePermissionARNs(delta, a, b)

			if got := delta.DifferentAt("Spec.PermissionARNs"); got != test.different {
				t.Errorf("DifferentAt(Spec.PermissionARNs) = %v, want %v", got, test.different)
			}
		})
	}
}

func TestComparePermissionARNsDoesNotMutateInputs(t *testing.T) {
	a := newResourceShareWithPermissions(nil)
	b := newResourceShareWithPermissions([]string{defaultPrefixListPermission, customSubnetPermission})

	comparePermissionARNs(ackcompare.NewDelta(), a, b)

	if a.ko.Spec.PermissionARNs != nil {
		t.Errorf("desired PermissionARNs mutated to %v", aws.ToStringSlice(a.ko.Spec.PermissionARNs))
	}
	if got := aws.ToStringSlice(b.ko.Spec.PermissionARNs); len(got) != 2 {
		t.Errorf("latest PermissionARNs mutated to %v", got)
	}
}

func TestCompareStringSlices(t *testing.T) {
	tests := []struct {
		name       string
		a          []string
		b          []string
		wantAdd    []string
		wantDelete []string
	}{
		{
			name:       "replacing an entry adds only the new one",
			a:          []string{"p2"},
			b:          []string{"p1"},
			wantAdd:    []string{"p2"},
			wantDelete: []string{"p1"},
		},
		{
			name:       "every addition is retained",
			a:          []string{"p1", "p2", "p3"},
			b:          []string{},
			wantAdd:    []string{"p1", "p2", "p3"},
			wantDelete: []string{},
		},
		{
			name:       "identical slices produce no changes",
			a:          []string{"p1", "p2"},
			b:          []string{"p2", "p1"},
			wantAdd:    []string{},
			wantDelete: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toAdd, toDelete := compareStringSlices(aws.StringSlice(test.a), aws.StringSlice(test.b))

			if !unorderedEqual(toAdd, test.wantAdd) {
				t.Errorf("toAdd = %v, want %v", toAdd, test.wantAdd)
			}
			if !unorderedEqual(toDelete, test.wantDelete) {
				t.Errorf("toDelete = %v, want %v", toDelete, test.wantDelete)
			}
		})
	}
}

func unorderedEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := append([]string{}, a...)
	sortedB := append([]string{}, b...)
	sort.Strings(sortedA)
	sort.Strings(sortedB)
	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}

func TestCheckDuplicatePermissionTypes(t *testing.T) {
	otherCustomSubnetPermission := "arn:aws:ram:us-west-2:123456789012:permission/other-subnet-permission"

	tests := []struct {
		name        string
		resolved    []resolvedPermission
		wantErr     bool
		wantErrType string
	}{
		{
			name:     "empty",
			resolved: nil,
		},
		{
			name: "distinct resource types",
			resolved: []resolvedPermission{
				{arn: customSubnetPermission, resourceType: "ec2:Subnet"},
				{arn: defaultPrefixListPermission, resourceType: "ec2:PrefixList"},
			},
		},
		{
			name: "same ARN repeated is not a conflict",
			resolved: []resolvedPermission{
				{arn: customSubnetPermission, resourceType: "ec2:Subnet"},
				{arn: customSubnetPermission, resourceType: "ec2:Subnet"},
			},
		},
		{
			name: "two permissions for one resource type is terminal",
			resolved: []resolvedPermission{
				{arn: customSubnetPermission, resourceType: "ec2:Subnet"},
				{arn: otherCustomSubnetPermission, resourceType: "ec2:Subnet"},
			},
			wantErr:     true,
			wantErrType: "ec2:Subnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDuplicatePermissionTypes(tt.resolved)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			var terminal *ackerr.TerminalError
			if !errors.As(err, &terminal) {
				t.Errorf("expected a TerminalError, got %T", err)
			}
			if !strings.Contains(err.Error(), tt.wantErrType) {
				t.Errorf("expected error to name resource type %q, got %q", tt.wantErrType, err.Error())
			}
		})
	}
}
