# RAM Controller E2E Tests

Integration tests for the AWS RAM (Resource Access Manager) ACK controller.

## Prerequisites

1. **Kubernetes cluster** with the RAM controller deployed
2. **AWS credentials** configured
3. **Python 3.8+** installed
4. **kubectl** configured to access your cluster

## Setup

Install test dependencies:

```bash
cd test/e2e
pip install -r requirements.txt
```

## Running Tests

### Option 1: With Controller Running in Cluster

If the RAM controller is already deployed to your cluster:

```bash
export AWS_PROFILE="your-aws-profile"
pytest tests/ -v
```

### Option 2: With Controller Running Locally

Terminal 1 - Start the controller:
```bash
export AWS_PROFILE="your-aws-profile"
cd /path/to/ram-controller
./bin/controller --aws-region us-east-2 --enable-development-logging
```

Terminal 2 - Run tests:
```bash
export AWS_PROFILE="your-aws-profile"
cd test/e2e
pytest tests/ -v
```

### Run Specific Tests

```bash
# Run only ResourceShare tests
pytest tests/test_resource_share.py -v

# Run only ResourceShareAccepter tests
pytest tests/test_resource_share_accepter.py -v

# Run only Permission tests
pytest tests/test_permission.py -v

# Run with specific markers
pytest -m canary -v
```

## Test Structure

- `tests/` - Test files
  - `test_resource_share.py` - ResourceShare CRUD tests
  - `test_resource_share_accepter.py` - ResourceShareAccepter tests
  - `test_permission.py` - Permission tests

- `resources/` - YAML templates for test resources
  - `ram_resource_share.yaml`
  - `ram_resource_share_accepter.yaml`
  - `ram_permission.yaml`

- Helper modules:
  - `ram_resource_share.py` - ResourceShare utilities
  - `ram_resource_share_accepter.py` - ResourceShareAccepter utilities
  - `ram_permission.py` - Permission utilities

## ResourceShareAccepter Tests

The `test_resource_share_accepter.py` tests validate the ResourceShareAccepter CRD.

### Current Tests

- **test_no_pending_invitation**: Tests error handling when no pending invitation exists

### Future Tests (Require Cross-Account Setup)

To test actual invitation acceptance, you need:
1. Two AWS accounts (sender and receiver)
2. Sender account creates a ResourceShare
3. Receiver account uses ResourceShareAccepter to accept

See comments in `test_resource_share_accepter.py` for implementation guidance.

## Troubleshooting

### "Controller did not consume the resource"

**Cause**: The RAM controller is not running or not watching the namespace.

**Solution**: 
- Verify controller is running: `kubectl get pods -n ack-system`
- Check controller logs: `kubectl logs -n ack-system deployment/ack-ram-controller`
- Or run controller locally (see Option 2 above)

### "Status not populated"

**Cause**: Controller hasn't reconciled the resource yet.

**Solution**: Wait longer or check controller logs for errors.

### AWS Credential Errors

**Cause**: AWS credentials not configured or expired.

**Solution**: 
```bash
export AWS_PROFILE="your-profile"
aws sts get-caller-identity  # Verify credentials work
```

