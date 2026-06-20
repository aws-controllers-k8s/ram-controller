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

"""Integration tests for the ResourceShareInvitation API.

NOTE: These tests require the RAM controller to be running in the cluster.
      The controller must be deployed before running these tests.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ram_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_KIND = "ResourceShareInvitation"
RESOURCE_PLURAL = "resourceshareinvitations"

CREATE_WAIT_AFTER_SECONDS = 15
DELETE_WAIT_AFTER_SECONDS = 10


@pytest.fixture(scope="module")
def resource_share_invitation_malformed_arn():
    """Creates a ResourceShareInvitation with a malformed ARN.

    This tests the terminal error handling path: a malformed ARN triggers
    MalformedArnException, which is listed in terminal_codes.
    """
    resource_name = random_suffix_name("share-invitation", 24)

    # A malformed ARN triggers MalformedArnException (a terminal code).
    malformed_arn = "arn:malformed:invitation"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["RESOURCE_SHARE_INVITATION_NAME"] = resource_name
    replacements["INVITATION_ARN"] = malformed_arn

    resource_data = load_ram_resource(
        "ram_resource_share_invitation",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    yield cr, ref

    try:
        k8s.delete_custom_resource(ref, period_length=DELETE_WAIT_AFTER_SECONDS)
    except:
        pass


@service_marker
@pytest.mark.canary
class TestResourceShareInvitation:
    def test_malformed_arn_terminal(self, resource_share_invitation_malformed_arn):
        """Test that ResourceShareInvitation sets a Terminal condition for a malformed ARN.

        This test validates:
        - Resource is created successfully
        - Controller processes the resource
        - Terminal condition is set when the ARN is malformed
        - Spec fields are preserved

        REQUIRES: RAM controller must be running in the cluster
        """
        res, ref = resource_share_invitation_malformed_arn

        assert res is not None, "Controller did not consume the resource. Is the RAM controller running?"

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'spec' in cr
        assert 'resourceShareInvitationARN' in cr['spec']

        assert 'status' in cr, "Status not populated. Controller may not be running or reconciliation failed."

        assert 'conditions' in cr['status']
        conditions = cr['status']['conditions']
        assert conditions is not None
        assert len(conditions) > 0

        terminal_condition = None
        for condition in conditions:
            if condition['type'] == 'ACK.Terminal':
                terminal_condition = condition
                break

        assert terminal_condition is not None, f"No Terminal condition found. Conditions: {conditions}"
        assert terminal_condition['status'] == 'True'

        logging.info(f"Terminal condition message: {terminal_condition['message']}")

        _, deleted = k8s.delete_custom_resource(
            ref,
            period_length=DELETE_WAIT_AFTER_SECONDS,
        )
        assert deleted


# NOTE: To test actual invitation acceptance, you need:
# 1. Two AWS accounts (sender and receiver)
# 2. Sender account creates a ResourceShare and shares with receiver account
# 3. Receiver account looks up the invitation ARN and creates a
#    ResourceShareInvitation CR referencing it
# 4. Verify invitation is accepted and status fields are populated
