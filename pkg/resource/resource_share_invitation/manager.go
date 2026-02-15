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

package resource_share_invitation

import (
	"context"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackcfg "github.com/aws-controllers-k8s/runtime/pkg/config"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=ram.services.k8s.aws,resources=resourceshareaccepters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ram.services.k8s.aws,resources=resourceshareaccepters/status,verbs=get;update;patch

// resourceManager is responsible for providing a consistent way to perform
// CRUD operations in a backend AWS service API for ResourceShareInvitation custom resources.
type resourceManager struct {
	// cfg is a copy of the ackcfg.Config object passed on start of the service
	// controller
	cfg ackcfg.Config
	// clientcfg is a copy of the client configuration passed on start of the
	// service controller
	clientcfg aws.Config
	// log refers to the logr.Logger object handling logging for the service
	// controller
	log logr.Logger
	// metrics contains a collection of Prometheus metric objects that the
	// service controller and its reconcilers track
	metrics *ackmetrics.Metrics
	// rr is the Reconciler which can be used for various utility
	// functions such as querying for Secret values given a SecretReference
	rr acktypes.Reconciler
	// awsAccountID is the AWS account identifier that contains the resources
	// managed by this resource manager
	awsAccountID ackv1alpha1.AWSAccountID
	// The AWS Region that this resource manager targets
	awsRegion ackv1alpha1.AWSRegion
	// sdk is a pointer to the AWS service API client exposed by the
	// aws-sdk-go-v2/services/ram package.
	sdkapi *svcsdk.Client
}

// concreteResource returns a pointer to a resource from the supplied
// generic AWSResource interface
func (rm *resourceManager) concreteResource(
	res acktypes.AWSResource,
) *resource {
	return res.(*resource)
}

// ReadOne returns the currently-observed state of the supplied AWSResource in
// the backend AWS service API.
func (rm *resourceManager) ReadOne(
	ctx context.Context,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		// Should never happen... if it does, it's buggy code.
		panic("resource manager's ReadOne() method received resource with nil CR object")
	}
	latest, err := rm.sdkFind(ctx, r)
	if err != nil {
		return nil, err
	}
	return latest, nil
}

// Create attempts to create the supplied AWSResource in the backend AWS
// service API, returning an AWSResource representing the newly-created
// resource
func (rm *resourceManager) Create(
	ctx context.Context,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		// Should never happen... if it does, it's buggy code.
		panic("resource manager's Create() method received resource with nil CR object")
	}
	created, err := rm.sdkCreate(ctx, r)
	if err != nil {
		if created != nil {
			return rm.onError(created, err)
		}
		return rm.onError(r, err)
	}
	return rm.onSuccess(created)
}

// Update attempts to mutate the supplied desired AWSResource in the backend AWS
// service API, returning an AWSResource representing the newly-mutated
// resource.
func (rm *resourceManager) Update(
	ctx context.Context,
	desired acktypes.AWSResource,
	latest acktypes.AWSResource,
	delta *ackcompare.Delta,
) (acktypes.AWSResource, error) {
	dres := rm.concreteResource(desired)
	lres := rm.concreteResource(latest)
	if dres.ko == nil || lres.ko == nil {
		// Should never happen... if it does, it's buggy code.
		panic("resource manager's Update() method received resource with nil CR object")
	}
	updated, err := rm.sdkUpdate(ctx, dres, lres, delta)
	if err != nil {
		return rm.onError(lres, err)
	}
	return rm.onSuccess(updated)
}

// Delete attempts to destroy the supplied AWSResource in the backend AWS
// service API, returning an AWSResource representing the
// resource being deleted (if delete is asynchronous and takes time)
func (rm *resourceManager) Delete(
	ctx context.Context,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		// Should never happen... if it does, it's buggy code.
		panic("resource manager's Delete() method received resource with nil CR object")
	}
	if err := rm.sdkDelete(ctx, r); err != nil {
		return rm.onError(r, err)
	}
	return r, nil
}




// ARNFromName returns an AWS ARN for a given resource name
func (rm *resourceManager) ARNFromName(name string) string {
	return ""
}

// LateInitialize is a no-op for ResourceShareInvitation
func (rm *resourceManager) LateInitialize(
	ctx context.Context,
	latest acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	return latest, nil
}

// EnsureTags is a no-op for ResourceShareAccepter (invitations don't have tags)
func (rm *resourceManager) EnsureTags(
	ctx context.Context,
	res acktypes.AWSResource,
	md acktypes.ServiceControllerMetadata,
) error {
	return nil
}

// IsSynced returns true if the resource is synced
func (rm *resourceManager) IsSynced(
	ctx context.Context,
	res acktypes.AWSResource,
) (bool, error) {
	return true, nil
}

// onError sets the Synced condition to False and returns the resource and error
func (rm *resourceManager) onError(
	r *resource,
	err error,
) (*resource, error) {
	errMsg := err.Error()
	ackcondition.SetSynced(r, corev1.ConditionFalse, &errMsg, nil)
	return r, err
}

// onSuccess sets the Synced condition to True and returns the resource
func (rm *resourceManager) onSuccess(
	r *resource,
) (*resource, error) {
	ackcondition.SetSynced(r, corev1.ConditionTrue, nil, nil)
	return r, nil
}

// newResourceManager returns a new resourceManager instance
func newResourceManager(
	cfg ackcfg.Config,
	clientcfg aws.Config,
	log logr.Logger,
	metrics *ackmetrics.Metrics,
	rr acktypes.Reconciler,
	id ackv1alpha1.AWSAccountID,
	region ackv1alpha1.AWSRegion,
) (*resourceManager, error) {
	return &resourceManager{
		cfg:          cfg,
		clientcfg:    clientcfg,
		log:          log,
		metrics:      metrics,
		rr:           rr,
		awsAccountID: id,
		awsRegion:    region,
		sdkapi:       svcsdk.NewFromConfig(clientcfg),
	}, nil
}

// GetRegion returns the AWS Region this resource manager is configured for
func (rm *resourceManager) GetRegion() ackv1alpha1.AWSRegion {
	return rm.awsRegion
}

// ResolveReferences is a no-op for ResourceShareAccepter
func (rm *resourceManager) ResolveReferences(
	ctx context.Context,
	apiReader client.Reader,
	res acktypes.AWSResource,
) (acktypes.AWSResource, bool, error) {
	return res, false, nil
}

// ClearResolvedReferences is a no-op for ResourceShareAccepter
func (rm *resourceManager) ClearResolvedReferences(
	res acktypes.AWSResource,
) acktypes.AWSResource {
	return res
}

// FilterSystemTags is a no-op for ResourceShareAccepter (invitations don't have tags)
func (rm *resourceManager) FilterSystemTags(res acktypes.AWSResource, systemTags []string) {
	// No-op: ResourceShareAccepter doesn't support tags
}
