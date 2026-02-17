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

"""Integration tests for the ResourceShare Accept Mode functionality.

Accept Mode allows a ResourceShare to accept an incoming share invitation
from another AWS account, rather than creating a new share.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ram_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_KIND = "ResourceShare"
RESOURCE_PLURAL = "resourceshares"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 20


@service_marker
class TestResourceShareAcceptMode:
    """Tests for ResourceShare Accept Mode functionality.
    
    Accept Mode is used when you want to accept a resource share invitation
    from another AWS account, rather than creating a new share.
    """

    def test_create_mode_name_validation(self):
        """Test that creating a ResourceShare in Create Mode without a name fails.
        
        When acceptInvitation is NOT set (Create Mode), the name field is required.
        This test verifies that the controller returns a Terminal error when name is missing.
        """
        resource_name = random_suffix_name("no-name-test", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["RESOURCE_SHARE_NAME"] = resource_name

        # Load ResourceShare CR without name field
        resource_data = load_ram_resource(
            "ram_resource_share_no_name",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Verify resource has Terminal error condition
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'conditions' in cr['status']
        
        # Check for ACK.Terminal condition
        terminal_condition = None
        for condition in cr['status']['conditions']:
            if condition['type'] == 'ACK.Terminal':
                terminal_condition = condition
                break
        
        assert terminal_condition is not None, "Expected ACK.Terminal condition"
        assert terminal_condition['status'] == 'True', "Expected Terminal condition to be True"
        assert 'name is required' in terminal_condition['message'].lower(), \
            f"Expected error message about name being required, got: {terminal_condition['message']}"
        
        logging.info(f"Terminal error message: {terminal_condition['message']}")

        # Clean up
        k8s.delete_custom_resource(ref, period_length=DELETE_WAIT_AFTER_SECONDS)

    @pytest.mark.skip(reason="Requires cross-account setup with actual invitation")
    def test_accept_mode_happy_path(self):
        """Test accepting a resource share invitation (Accept Mode).
        
        This test requires:
        1. Another AWS account (Account A) to create a resource share
        2. An invitation sent to the test account (Account B)
        3. The ShareARN of the invitation
        
        TODO: Implement this test when cross-account test infrastructure is available.
        
        Expected behavior:
        - ResourceShare is created with acceptInvitation field
        - Controller finds the pending invitation
        - Controller accepts the invitation
        - Status fields are populated: invitationARN, invitationStatus, senderAccountID, etc.
        - Invitation status becomes ACCEPTED
        """
        pass

    @pytest.mark.skip(reason="Requires cross-account setup with actual invitation")
    def test_accept_mode_deletion(self):
        """Test deleting a ResourceShare in Accept Mode.
        
        This test requires:
        1. An accepted resource share invitation
        2. A ResourceShare in Accept Mode that has accepted the invitation
        
        TODO: Implement this test when cross-account test infrastructure is available.
        
        Expected behavior:
        - When deleting a ResourceShare with ACCEPTED invitation:
          - Controller calls DisassociateResourceShare (leaves the share)
          - Invitation is cleaned up in AWS
          - Kubernetes resource is deleted
        """
        pass

    @pytest.mark.skip(reason="Requires cross-account setup with actual invitation")
    def test_accept_mode_nonexistent_invitation(self):
        """Test accepting a non-existent invitation.
        
        This test verifies error handling when trying to accept an invitation
        that doesn't exist.
        
        TODO: Implement this test when cross-account test infrastructure is available.
        
        Expected behavior:
        - ResourceShare is created with acceptInvitation pointing to non-existent share
        - Controller returns Terminal error
        - Error message indicates no pending invitation found
        """
        pass

