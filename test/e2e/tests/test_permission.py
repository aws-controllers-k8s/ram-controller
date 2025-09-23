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

"""Integration tests for the Permission API.
"""

import pytest
import time
import json
import logging
import boto3

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ram_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import ram_permission

RESOURCE_PLURAL = "permissions"

CREATE_WAIT_AFTER_SECONDS = 5
MODIFY_WAIT_AFTER_SECONDS = 31
DELETE_WAIT_AFTER_SECONDS = 20

@pytest.fixture(scope="module")
def permission():
	resource_name = random_suffix_name("permission", 24)

	resources = get_bootstrap_resources()
	logging.debug(resources)

	replacements = REPLACEMENT_VALUES.copy()
	replacements["PERMISSION_NAME"] = resource_name

	# Load Permission CR
	resource_data = load_ram_resource(
		"ram_permission",
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
	time.sleep(CREATE_WAIT_AFTER_SECONDS)
	cr = k8s.get_resource(ref)
	yield cr, ref

	# Delete k8s resource
	_, deleted = k8s.delete_custom_resource(
		ref,
		period_length=DELETE_WAIT_AFTER_SECONDS,
	)
	assert deleted

@service_marker
@pytest.mark.canary
class TestPermission:
	def test_crud(self, permission):
		res, ref = permission

		# Verify state and spec have values we need to test
		cr = k8s.get_resource(ref)
		assert cr is not None
		assert 'spec' in cr
		assert 'policyTemplate' in cr['spec']
		assert 'status' in cr
		assert 'status' in cr['status']
		assert cr['status']['status'] == "ATTACHABLE"
		assert 'version' in cr['status']
		assert cr['status']['version'] == '1'
		assert 'ackResourceMetadata' in cr['status']
		assert 'arn' in cr['status']['ackResourceMetadata']
		resource_arn = cr['status']['ackResourceMetadata']['arn']

		# Test updating Permission by changing policyTemplate
		original_policy_template = {
			"Action": ["imagebuilder:ListComponents"],
			"Effect": "Allow"
		}
		permission_version = ram_permission.get_permission(resource_arn, 1)
		policy_template = permission_version["permission"]
		assert policy_template == json.dumps(original_policy_template)
		assert permission_version["defaultVersion"]

		updated_policy_template = {
			'Action': ['imagebuilder:ListComponents', 'imagebuilder:GetComponent'],
			'Effect': 'Allow'
		}
		updates = {
			"spec": {
				"policyTemplate": json.dumps(updated_policy_template)
			}
		}
		k8s.patch_custom_resource(ref, updates)
		time.sleep(MODIFY_WAIT_AFTER_SECONDS)

		# Check resource synced successfully
		assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

		cr = k8s.get_resource(ref)
		assert 'spec' in cr
		assert 'policyTemplate' in cr['spec']
		assert 'status' in cr
		assert 'version' in cr['status']
		assert cr['status']['version'] == '2'
		assert 'defaultVersion' in cr['status']
		assert cr['status']['defaultVersion']

		permission_version = ram_permission.get_permission(resource_arn, 2)
		policy_template = permission_version["permission"]
		assert policy_template == json.dumps(updated_policy_template)
		assert permission_version["defaultVersion"]

		# Test reverting back to previous policyTemplate
		updates = {
			"spec": {
				"policyTemplate": json.dumps(original_policy_template)
			}
		}
		k8s.patch_custom_resource(ref, updates)
		time.sleep(MODIFY_WAIT_AFTER_SECONDS)

		# Check resource synced successfully
		assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
		
		cr = k8s.get_resource(ref)
		assert 'spec' in cr
		assert 'policyTemplate' in cr['spec']
		assert 'status' in cr
		assert 'version' in cr['status']
		assert cr['status']['version'] == '3'
		assert 'defaultVersion' in cr['status']
		assert cr['status']['defaultVersion']


		permission_version = ram_permission.get_permission(resource_arn, 3)	
		policy_template = permission_version["permission"]
		assert policy_template == json.dumps(original_policy_template)
		assert permission_version["defaultVersion"]

		# Check how many permissionVersions we have for permissionArn
		pvs = ram_permission.list_permission_versions(resource_arn)
		attachable_permissions = 0
		for pv in pvs:
			if pv['status'] == "ATTACHABLE":
				attachable_permissions += 1
		
		assert attachable_permissions == 1


		# Test updating ResourceShare by adding tags
		user_tag = {
			"key": "my-key",
			"value": "my_val"
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
		assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

		latest = ram_permission.get_permission(resource_arn)
		assert 'tags' in latest
		assert user_tag in latest['tags']
