# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Utilities for working with RAM ResourceShareAccepter resources"""

import datetime
import time

import boto3
import pytest

DEFAULT_WAIT_TIMEOUT_SECONDS = 120
DEFAULT_WAIT_INTERVAL_SECONDS = 15


def get_resource_share_invitation(share_arn: str):
    """Returns the ResourceShareInvitation for a supplied share ARN.

    If no pending invitation exists, returns None.
    """
    c = boto3.client('ram')
    try:
        resp = c.get_resource_share_invitations(
            resourceShareArns=[share_arn]
        )
        invitations = resp.get('resourceShareInvitations', [])
        # Return the first pending invitation for this share
        for invitation in invitations:
            if invitation.get('status') == 'PENDING':
                return invitation
        return None
    except Exception as e:
        return None


def wait_until_invitation_accepted(
        share_arn: str,
        timeout_seconds: int = DEFAULT_WAIT_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a ResourceShareInvitation for the supplied share ARN is ACCEPTED.

    Usage:
        from e2e.ram_resource_share_accepter import wait_until_invitation_accepted

        wait_until_invitation_accepted(share_arn)

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for ResourceShareInvitation to be ACCEPTED"
            )
        time.sleep(interval_seconds)

        invitation = get_resource_share_invitation(share_arn)
        if invitation and invitation.get('status') == 'ACCEPTED':
            break

