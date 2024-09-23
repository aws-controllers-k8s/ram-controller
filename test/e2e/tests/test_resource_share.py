# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the ResourceShare API.
"""

import pytest
import time
import logging
import boto3
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ram_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import ram_resource_share, ram_permission

RESOURCE_KIND = "ResourceShare"
RESOURCE_PLURAL = "resourceshares"

CREATE_WAIT_AFTER_SECONDS = 5
MODIFY_WAIT_AFTER_SECONDS = 20
DELETE_WAIT_AFTER_SECONDS = 20


@pytest.fixture(scope="module")
def resource_share():
    resource_name = random_suffix_name("resource-share", 24)

    resources = get_bootstrap_resources()
    logging.debug(resources)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["RESOURCE_SHARE_NAME"] = resource_name

    # Load ResourceShare CR
    resource_data = load_ram_resource(
        "ram_resource_share",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    ram_resource_share.wait_until_exists(resource_name)

    yield cr, ref

@pytest.fixture(scope="module")
def resource_share_association():
    resource_name = random_suffix_name("resource-share", 24)

    subnet = get_bootstrap_resources().RamVPC.public_subnets
    subnet_arn = f"arn:aws:ec2:{subnet.region}:{subnet.account_id}:subnet/{subnet.subnet_ids[0]}"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["RESOURCE_SHARE_NAME"] = resource_name
    replacements["RESOURCE_ARN"] = subnet_arn

    # Load ResourceShare CR
    resource_data = load_ram_resource(
        "ram_resource_share_association",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    ram_resource_share.wait_until_exists(resource_name)

    yield cr, ref

@service_marker
@pytest.mark.canary
class TestResourceShare:
    def test_crud(self, resource_share):
        res, ref = resource_share

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Verify state and spec have values we need to test
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'spec' in cr
        assert 'allowExternalPrincipals' in cr['spec']
        assert cr['spec']['allowExternalPrincipals'] == True
        assert 'status' in cr
        resource_name = cr['spec']['name']

        # Test updating ResourceShare by adding tags
        user_tag = {
                "key": "someKey",
                "value": "someValue",
            }
        updates = {
            "spec": {
                "tags":
                    [user_tag]
            }
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        latest = ram_resource_share.get_resource_shares(resource_share_name=resource_name)
        assert 'tags' in latest
        assert user_tag in latest['tags']
        assert 'resourceShareArn' in latest
        
        resource_share_arn = latest['resourceShareArn']

        # Test associating Permission refs
        permission_name = random_suffix_name("permission-name", 24)
        test_values = REPLACEMENT_VALUES.copy()
        test_values['PERMISSION_NAME'] = permission_name

        permission_resource_data = load_ram_resource(
            "ram_permission",
            additional_replacements=test_values,
        )

        permission_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'permissions',
            permission_name, namespace="default"
        )
        k8s.create_custom_resource(permission_ref, permission_resource_data)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(permission_ref, "ACK.ResourceSynced", "True", wait_periods=5)

        permission_reference = {
            "from": {
                "name": permission_name
            }
        }

        updates = {
            "spec": {
                "permissionRefs": [permission_reference]
            }
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        associated_permissions = ram_resource_share.list_associated_permissions(arn=resource_share_arn)

        permission_arn = associated_permissions[0]['arn']

        p_name = permission_arn.split("permission/")

        assert len(p_name) > 1

        assert p_name[1] == permission_name

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(
            ref,
            period_length=DELETE_WAIT_AFTER_SECONDS,
        )
        assert deleted

        _, deleted = k8s.delete_custom_resource(permission_ref, period_length=DELETE_WAIT_AFTER_SECONDS)
        assert deleted

    def test_resource_association(self, resource_share_association):

        res, ref = resource_share_association

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Verify state and spec have values we need to test
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'spec' in cr
        assert 'allowExternalPrincipals' in cr['spec']
        assert cr['spec']['allowExternalPrincipals'] == True
        assert 'resourceARNs' in cr['spec']
        assert len(cr['spec']['resourceARNs']) == 1
        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] == "ACTIVE"
        
        subnet_arn = cr['spec']['resourceARNs'][0]
        resource_share_arn = cr['status']['ackResourceMetadata']['arn']

        associated_resource = ram_resource_share.list_associated_resources(arn=resource_share_arn)

        assert 'status' in associated_resource
        assert associated_resource['status'] == "ASSOCIATED"
        assert associated_resource['associatedEntity'] == subnet_arn

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(
            ref,
            period_length=DELETE_WAIT_AFTER_SECONDS,
        )
        assert deleted
