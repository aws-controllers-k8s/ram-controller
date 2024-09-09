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

package resource_share

import (
	"bytes"
	"reflect"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
)

// Hack to avoid import errors during build...
var (
	_ = &bytes.Buffer{}
	_ = &reflect.Method{}
	_ = &acktags.Tags{}
)

// newResourceDelta returns a new `ackcompare.Delta` used to compare two
// resources
func newResourceDelta(
	a *resource,
	b *resource,
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
	if (a == nil && b != nil) ||
		(a != nil && b == nil) {
		delta.Add("", a, b)
		return delta
	}
	compareTags(delta, a, b)

	if ackcompare.HasNilDifference(a.ko.Spec.AllowExternalPrincipals, b.ko.Spec.AllowExternalPrincipals) {
		delta.Add("Spec.AllowExternalPrincipals", a.ko.Spec.AllowExternalPrincipals, b.ko.Spec.AllowExternalPrincipals)
	} else if a.ko.Spec.AllowExternalPrincipals != nil && b.ko.Spec.AllowExternalPrincipals != nil {
		if *a.ko.Spec.AllowExternalPrincipals != *b.ko.Spec.AllowExternalPrincipals {
			delta.Add("Spec.AllowExternalPrincipals", a.ko.Spec.AllowExternalPrincipals, b.ko.Spec.AllowExternalPrincipals)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.Name, b.ko.Spec.Name) {
		delta.Add("Spec.Name", a.ko.Spec.Name, b.ko.Spec.Name)
	} else if a.ko.Spec.Name != nil && b.ko.Spec.Name != nil {
		if *a.ko.Spec.Name != *b.ko.Spec.Name {
			delta.Add("Spec.Name", a.ko.Spec.Name, b.ko.Spec.Name)
		}
	}
	if len(a.ko.Spec.PermissionARNs) != len(b.ko.Spec.PermissionARNs) {
		delta.Add("Spec.PermissionARNs", a.ko.Spec.PermissionARNs, b.ko.Spec.PermissionARNs)
	} else if len(a.ko.Spec.PermissionARNs) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.PermissionARNs, b.ko.Spec.PermissionARNs) {
			delta.Add("Spec.PermissionARNs", a.ko.Spec.PermissionARNs, b.ko.Spec.PermissionARNs)
		}
	}
	if len(a.ko.Spec.Principals) != len(b.ko.Spec.Principals) {
		delta.Add("Spec.Principals", a.ko.Spec.Principals, b.ko.Spec.Principals)
	} else if len(a.ko.Spec.Principals) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.Principals, b.ko.Spec.Principals) {
			delta.Add("Spec.Principals", a.ko.Spec.Principals, b.ko.Spec.Principals)
		}
	}
	if len(a.ko.Spec.ResourceARNs) != len(b.ko.Spec.ResourceARNs) {
		delta.Add("Spec.ResourceARNs", a.ko.Spec.ResourceARNs, b.ko.Spec.ResourceARNs)
	} else if len(a.ko.Spec.ResourceARNs) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.ResourceARNs, b.ko.Spec.ResourceARNs) {
			delta.Add("Spec.ResourceARNs", a.ko.Spec.ResourceARNs, b.ko.Spec.ResourceARNs)
		}
	}
	if len(a.ko.Spec.Sources) != len(b.ko.Spec.Sources) {
		delta.Add("Spec.Sources", a.ko.Spec.Sources, b.ko.Spec.Sources)
	} else if len(a.ko.Spec.Sources) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.Sources, b.ko.Spec.Sources) {
			delta.Add("Spec.Sources", a.ko.Spec.Sources, b.ko.Spec.Sources)
		}
	}

	return delta
}
