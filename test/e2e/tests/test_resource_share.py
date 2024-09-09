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

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ram_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import ram_resource_share

RESOURCE_KIND = "ResourceShare"
RESOURCE_PLURAL = "resourceshares"

CREATE_WAIT_AFTER_SECONDS = 5
MODIFY_WAIT_AFTER_SECONDS = 10
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

    # Delete k8s resource
    _, deleted = k8s.delete_custom_resource(
        ref,
        period_length=DELETE_WAIT_AFTER_SECONDS,
    )
    assert deleted

    ram_resource_share.wait_until_deleted(resource_name)


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
