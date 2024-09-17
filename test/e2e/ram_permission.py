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
        permission_arn: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_EXISTS_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_EXISTS_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a Permission with a supplied ARN is returned from RAM 
    GetPermission API.

    Usage:
        from e2e.ram_permission import wait_until_exists

        wait_until_exists()

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for Permission to exist "
                "in RAM API"
            )
        time.sleep(interval_seconds)

        latest = get_permission(permission_arn)
        if latest != None and latest['status'] == "ATTACHABLE":
            break


def wait_until_deleted(
        permission_arn: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a Permission with a supplied ARN is returned with
    "DELETED" status from the RAM API.

    Usage:
        from e2e.ram_permission import wait_until_deleted

        wait_until_deleted(queue_name)

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for Permission to be "
                "deleted in RAM API"
            )
        time.sleep(interval_seconds)

        latest = get_permission(permission_arn)
        if latest != None and latest['status'] == "DELETED":
            break


def get_permission(permission_arn, permission_version=0):
    """Returns the Permission for a supplied Permission ARN.

    If no such Permission exists, returns None.
    """
    c = boto3.client('ram')
    try:
        resp = c.get_permission(permissionArn=permission_arn, permissionVersion=permission_version)
        if 'permission' in resp:
            return resp['permission']
        else:
            return None
    except c.exceptions.UnknownResourceException:
        return None

def list_permission_versions(permission_arn):
    """Returns a list of PermissionVersions for a supplied Permission ARN.

    If no such PermissionVersion exists, return None.
    """
    c = boto3.client('ram')
    try:
        resp = c.list_permission_versions(permissionArn=permission_arn)
        if 'permissions' in resp:
            return resp['permissions']
        else:
            return None
    except c.exceptions.UnknownResourceException:
        return None
