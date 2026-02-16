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

"""Integration tests for the ResourceShareAccepter API.

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

RESOURCE_KIND = "ResourceShareAccepter"
RESOURCE_PLURAL = "resourceshareaccepters"

CREATE_WAIT_AFTER_SECONDS = 15
DELETE_WAIT_AFTER_SECONDS = 10


@pytest.fixture(scope="module")
def resource_share_accepter_no_invitation():
    """Creates a ResourceShareAccepter for a non-existent share.

    This tests the error handling path when no pending invitation exists.
    """
    resource_name = random_suffix_name("share-accepter", 24)

    # Use a properly formatted but non-existent share ARN
    # This ARN format is valid but the UUID doesn't exist, so there will be no invitation
    fake_share_arn = "arn:aws:ram:us-east-2:065002218531:resource-share/ffffffff-ffff-ffff-ffff-ffffffffffff"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["RESOURCE_SHARE_ACCEPTER_NAME"] = resource_name
    replacements["SHARE_ARN"] = fake_share_arn
    
    # Load ResourceShareAccepter CR
    resource_data = load_ram_resource(
        "ram_resource_share_accepter",
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
    
    yield cr, ref
    
    # Cleanup
    try:
        k8s.delete_custom_resource(ref, period_length=DELETE_WAIT_AFTER_SECONDS)
    except:
        pass


@service_marker
@pytest.mark.canary
class TestResourceShareAccepter:
    def test_no_pending_invitation(self, resource_share_accepter_no_invitation):
        """Test that ResourceShareAccepter handles 'no pending invitation' gracefully.

        This test validates:
        - Resource is created successfully
        - Controller processes the resource
        - Terminal condition is set when no invitation exists
        - Spec fields are preserved

        REQUIRES: RAM controller must be running in the cluster
        """
        res, ref = resource_share_accepter_no_invitation

        # Verify the controller consumed the resource
        assert res is not None, "Controller did not consume the resource. Is the RAM controller running?"

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Get the resource
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'spec' in cr
        assert 'shareARN' in cr['spec']

        # Verify spec is preserved
        share_arn = cr['spec']['shareARN']
        assert share_arn.startswith("arn:aws:ram:")

        # Verify status exists
        assert 'status' in cr, "Status not populated. Controller may not be running or reconciliation failed."

        # Verify conditions exist and contain terminal state
        # The controller should set a Terminal condition when no invitation is found
        assert 'conditions' in cr['status']
        conditions = cr['status']['conditions']
        assert conditions is not None
        assert len(conditions) > 0

        # Check for Terminal condition
        terminal_condition = None
        for condition in conditions:
            if condition['type'] == 'ACK.Terminal':
                terminal_condition = condition
                break

        assert terminal_condition is not None, f"No Terminal condition found. Conditions: {conditions}"
        assert terminal_condition['status'] == 'True'
        # The message should indicate no invitation was found
        assert 'invitation' in terminal_condition['message'].lower() or 'not found' in terminal_condition['message'].lower()

        logging.info(f"✓ Terminal condition message: {terminal_condition['message']}")

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(
            ref,
            period_length=DELETE_WAIT_AFTER_SECONDS,
        )
        assert deleted


# NOTE: To test actual invitation acceptance, you need:
# 1. Two AWS accounts (sender and receiver)
# 2. Sender account creates a ResourceShare and shares with receiver account
# 3. Receiver account creates ResourceShareAccepter with the share ARN
# 4. Verify invitation is accepted and status fields are populated
#
# Example test structure (requires cross-account setup):
#
# @pytest.fixture(scope="module")
# def resource_share_accepter_with_invitation():
#     # This would require:
#     # - Creating a share in sender account
#     # - Getting the share ARN
#     # - Creating ResourceShareAccepter in receiver account
#     pass
#
# def test_accept_invitation(self, resource_share_accepter_with_invitation):
#     # Verify invitation is accepted
#     # Verify status fields: invitationARN, invitationStatus, senderAccountID, etc.
#     pass

