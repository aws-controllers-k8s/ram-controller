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

"""Utilities for working with RAM resources"""

import datetime
import json
import time

import boto3
import pytest

DEFAULT_WAIT_UNTIL_EXISTS_TIMEOUT_SECONDS = 120*10
DEFAULT_WAIT_UNTIL_EXISTS_INTERVAL_SECONDS = 15
DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 120*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15


def wait_until_exists(
        resource_share_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_EXISTS_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_EXISTS_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a ResourceShare with a supplied name is returned from RAM 
    GetResourceShares API.

    Usage:
        from e2e.ram_resource_share import wait_until_exists

        wait_until_exists()

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for ResourceShare to exist "
                "in RAM API"
            )
        time.sleep(interval_seconds)

        latest = get_resource_shares(resource_share_name)
        if latest != None and latest['status'] == "ACTIVE":
            break


def wait_until_deleted(
        resource_share_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a ResourceShare with a supplied name is returned with
    "DELETED" status from the RAM API.

    Usage:
        from e2e.queue import wait_until_deleted

        wait_until_deleted(queue_name)

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for ResourceShare to be "
                "deleted in RAM API"
            )
        time.sleep(interval_seconds)

        latest = get_resource_shares(resource_share_name)
        if latest != None and latest['status'] == "DELETED":
            break


def get_resource_shares(resource_share_name):
    """Returns the ResourceShare for a supplied ResourceShare name.

    If no such ResourceShare exists, returns None.
    """
    c = boto3.client('ram')
    try:
        resp = c.get_resource_shares(name=resource_share_name, resourceOwner="SELF")
        resource_shares = resp['resourceShares']
        for r in resource_shares:
            if r["name"] == resource_share_name:
                return r
    except c.exceptions.UnknownResourceException:
        return None
