# Uptime Service Backend

[![Build](https://github.com/o1-labs/uptime-service-backend/actions/workflows/build.yml/badge.svg)](https://github.com/o1-labs/uptime-service-backend/actions/workflows/build.yml)
[![Integration](https://github.com/o1-labs/uptime-service-backend/actions/workflows/integration.yml/badge.svg)](https://github.com/o1-labs/uptime-service-backend/actions/workflows/integration.yml)

As part of delegation program, nodes are to upload some proof of their activity. These proofs are to be accumulated and utilized for scoring. This service provides the nodes with a way to submit their data for score calculation.

## Constants

- `MAX_SUBMIT_PAYLOAD_SIZE` : max size (in bytes) of the `POST /submit` payload
- `REQUESTS_PER_PK_HOURLY` : max amount of requests per hour per public key `submitter` [default: 120, can be overriden by setting `REQUESTS_PER_PK_HOURLY` env variable].

## Protocol

1. Node submits a payload to the Service using `POST /submit`
2. Server saves the request
3. Server replies with status `ok` and HTTP 200 if payload is correct and some other HTTP code otherwise

## Interface

Backend Service is a web server that exposes the following entrypoints:

- `POST /submit` to submit a JSON payload containing the following data:

    ```json
    { "data":
       { "peer_id": "<base58-encoded peer id of the node from libp2p library>"
       , "block": "<base64-encoded bytes of the latest known block>"
       , "created_at": "<current time>"

       // Optional argument
       , "snark_work": "<base64-encoded snark work blob>"
       }
    , "submitter": "<base58check-encoded public key of the submitter>"
    , "sig": "<base64-encoded signature of `data` contents made with public key submitter above>"
    }
    ```

    - Mina's signature scheme (as described in [https://github.com/MinaProtocol/c-reference-signer](https://github.com/MinaProtocol/c-reference-signer)) is to be used
    - Time is represented according to `RFC-3339` with mandatory `Z` suffix (i.e. in UTC), like: `1985-04-12T23:20:50.52Z`
    - Payload for signing is to be made as the following JSON (it's important that its fields are in lexicographical order and if no `snark_work` is provided, field is omitted):
       - `block`: Base64 representation of a `block` field from payload
       - `created_at`: same as in `data`
       - `peer_id`: same as in `data`
       - `snark_work`: same as in `data` (omitted if `null` or `""`)
    - There are three possible responses:
        - `400 Bad Request` with `{"error": "<machine-readable description of an error>"}` payload when the input is considered malformed
        - `401 Unauthorized`  when public key `submitter` is not on the list of allowed keys or the signature is invalid
        - `411 Length Required` when no length header is provided
        - `413 Payload Too Large` when payload exceeds `MAX_SUBMIT_PAYLOAD_SIZE` constant
        - `429 Too Many Requests` when submission from public key `submitter` is rejected due to rate-limiting policy
        - `500 Internal Server Error` with `{"error": "<machine-readable description of an error>"}` payload for any other server error
        - `503 Service Unavailable` when IP-based rate-limiting prohibits the request
        - `200` with `{"status": "ok"}`

## Configuration

The program can be configured using either a JSON configuration file or environment variables. Below is the comprehensive guide on how to configure each option.

### Configuration Using a JSON File

1. **Set Configuration File Path**:
   Set the environment variable `CONFIG_FILE` to the path of your JSON configuration file.

2. **JSON Configuration Structure**:
   Your JSON file should adhere to the structure specified by the `AppConfig` struct in Go. Here is an example structure:

```json
{
  "network_name": "your_network_name",
  "gsheet_id": "your_google_sheet_id",
  "delegation_whitelist_list": "your_whitelist_list",
  "delegation_whitelist_column": "your_whitelist_column",
  "delegation_whitelist_disabled": false,
  // available storage configurations
  "aws": {
    "account_id": "your_aws_account_id",
    "bucket_name_suffix": "your_bucket_name_suffix",
    "region": "your_aws_region",
    "access_key_id": "your_aws_access_key_id",
    "secret_access_key": "your_aws_secret_access_key"
  },
  "aws_keyspaces": {
    "keyspace": "your_aws_keyspace",
    "region": "your_aws_region",
    "access_key_id": "your_aws_access_key_id",
    "secret_access_key": "your_aws_secret_access_key",
    "ssl_certificate_path": "your_aws_ssl_certificate_path"
  },
  "filesystem": {
    "path": "your_filesystem_path"
  },
  "postgresql": {
    "user": "postgres",
    "password": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "delegation_program",
    "sslmode": "require"
  }
}
```

### Configuration Using Environment Variables

If the `CONFIG_FILE` environment variable is not set, the program will fall back to loading configuration from environment variables.

1. **General Configuration**:
   - `CONFIG_NETWORK_NAME` - Set this to your network name.

2. **Whitelist Configuration**:
   - `GOOGLE_APPLICATION_CREDENTIALS` - set path to `minasheets.json` file including credentials to connect to Google Sheets.
   - `CONFIG_GSHEET_ID` - Set this to your Google Sheet ID with the keys to whitelist.
   - `DELEGATION_WHITELIST_LIST` - Set this to your delegation whitelist sheet title where the whitelist keys are.
   - `DELEGATION_WHITELIST_COLUMN` - Set this to your delegation whitelist sheet column where the whitelist keys are.
   - `DELEGATION_WHITELIST_REFRESH_INTERVAL` - Whitelist refresh interval in minutes. If not set default value `10` is used.
   -  Or disable whitelisting alltogether by setting `DELEGATION_WHITELIST_DISABLED=1`. The previous env variables are then ignored.

3. **AWS S3 Configuration**:
   - `AWS_ACCOUNT_ID` - Your AWS Account ID.
   - `AWS_BUCKET_NAME_SUFFIX` - Suffix for the AWS S3 bucket name.
   - `AWS_REGION` - The AWS region.
   - `AWS_ACCESS_KEY_ID` - Your AWS Access Key ID.
   - `AWS_SECRET_ACCESS_KEY` - Your AWS Secret Access Key.

4. **AWS Keyspaces/Cassandra Configuration**:

   **Mandatory/common env vars:**
   - `AWS_KEYSPACE` - Your Keyspace name.
   - `AWS_SSL_CERTIFICATE_PATH` - The path to your SSL certificate.

   **Depending on way of connecting:**
   
   _Service level connection:_
   - `CASSANDRA_HOST` - Cassandra host (e.g. cassandra.us-west-2.amazonaws.com).
   - `CASSANDRA_PORT` - Cassandra port (e.g. 9142).
   - `CASSANDRA_USERNAME` - Cassandra service user.
   - `CASSANDRA_PASSWORD` - Cassandra service password.
   
   _AWS access key / web identity token:_
   - `AWS_REGION` - The AWS region (same as used for S3).
   - `AWS_WEB_IDENTITY_TOKEN_FILE` - AWS web identity token file.
   - `AWS_ROLE_SESSION_NAME` - AWS role session name.
   - `AWS_ROLE_ARN` - AWS role arn.
   - `AWS_ACCESS_KEY_ID` - Your AWS Access Key ID. No need to set if `AWS_WEB_IDENTITY_TOKEN_FILE`, `AWS_ROLE_SESSION_NAME` and `AWS_ROLE_ARN` are set.
   - `AWS_SECRET_ACCESS_KEY` - Your AWS Secret Access Key. No need to set if `AWS_WEB_IDENTITY_TOKEN_FILE`, `AWS_ROLE_SESSION_NAME` and `AWS_ROLE_ARN` are set.

> **Note:** Docker image already includes cert and has `AWS_SSL_CERTIFICATE_PATH` set up, however it can be overriden by providing this env variable to docker.

5. **Local File System Configuration**:
   - `CONFIG_FILESYSTEM_PATH` - Set this to the path where you want the local file system to point.

6. **PostgreSQL Configuration**

If this storage backend is configured it is assumed that submissions are written into `submissions` table in the uptime-service-validation (coordinator) component. In this mode we are not storing `raw_block` in the database.

- `POSTGRES_HOST` - Hostname or IP address where your PostgreSQL server is running.
- `POSTGRES_PORT` - Port number on which PostgreSQL is listening.
- `POSTGRES_DB` - The name of the database to connect to. This is the uptime-service-validation database.
- `POSTGRES_USER` - The username with which to connect to the database.
- `POSTGRES_PASSWORD` - The password for the database user.
- `POSTGRES_SSLMODE` - The mode for SSL connectivity (e.g., `disable`, `require`, `verify-ca`, `verify-full`). Default is `require` for secure setups.

7. **Test settings**

These settings are useful for debugging or testing under controlled conditions. Always revert to secure and sensible defaults before moving to a production environment to maintain the security and reliability of your system.

 - `VERIFY_SIGNATURE_DISABLED` - set to `1` to disable signature verification on submission. It is `0` by default.
 - `REQUESTS_PER_PK_HOURLY` - set to arbitrarily high value if you want more requests accepted from a single submitter per hour. Default is `120`. 

### Important Notes

- At least one of the following storage options is required: `AwsS3`, `AwsKeyspaces`, or `LocalFileSystem`. Multi-storage configuration is also supported, allowing for a combination of these storage options.
- Ensure that all necessary environment variables are set. If any required variable is missing, the program will terminate with an error.

### Database Migration

When using `AWSKeyspaces` as storage for the first time one needs to run database migration script in order to create necessary tables. After `AWSKeyspaces` config is properly set on the environment, one can run database migration using the provided script:

```bash
$ nix-shell
# To migrate database up
[nix-shell]$ make db-migrate-up

# To migrate database down
[nix-shell]$ make db-migrate-down
```

Migration is also possible from dockerfile using non-default entrypoint `db_migration` for instance:

```bash
# To migrate database up
docker run \
-e AWS_KEYSPACE=keyspace_name \
-e AWS_REGION=us-west-2 \
-e AWS_ACCESS_KEY_ID=*** \
-e AWS_SECRET_ACCESS_KEY=*** \
-e DELEGATION_WHITELIST_DISABLED=1 \
-e CONFIG_NETWORK_NAME=integration-test \
--entrypoint db_migration \
673156464838.dkr.ecr.us-west-2.amazonaws.com/uptime-service-backend:$TAG up

# To migrate database down
docker run \
-e AWS_KEYSPACE=keyspace_name \
-e AWS_REGION=us-west-2 \
-e AWS_ACCESS_KEY_ID=*** \
-e AWS_SECRET_ACCESS_KEY=*** \
-e DELEGATION_WHITELIST_DISABLED=1 \
-e CONFIG_NETWORK_NAME=integration-test \
--entrypoint db_migration \
673156464838.dkr.ecr.us-west-2.amazonaws.com/uptime-service-backend:$TAG down
```

Once you have set up your configuration using either a JSON file or environment variables, you can proceed to run the program. The program will automatically load the configuration and initialize based on the provided settings.

## Storage

Based on storage option used in config the program will store blocks and submissions either in local filesystem, AWS S3 or Cassandra database in AWS Keyspaces.

The AWS S3 and filesystem storage has the following structure:

- `submissions`
    - `<submitted_at_date>/<submitted_at>-<submitter>.json`
      - Path contents:
        - `submitted_at_date` with server's date (of the time of submission) in format `YYYY-MM-DD`
        - `submitted_at` with server's timestamp (of the time of submission) in RFC-3339
        - `submitter` is base58check-encoded submitter's public key
      - File contents:
        - `remote_addr` with the `ip:port` address from which request has come
        - `peer_id` (as in user's JSON submission)
        - `snark_work` (optional, as in user's JSON submission)
        - `submitter` is base58check-encoded submitter's public key
        - `created_at` is UTC-based `RFC-3339` -encoded
        - `block_hash` is base58check-encoded hash of a block
- `blocks`
    - `<block-hash>.dat`
        - Contains raw block

In case of AWS Keyspaces the storage is kept in two tables `blocks` and `submissions`. The structure of the tables can be found in [/database/migrations](/database/migrations).

## Validation and rate limitting

All endpoints are guarded with Nginx which acts as a:

- HTTPS proxy
- Rate-limiter (by IP address) configured with `REQUESTS_PER_IP_HOURLY`

On receiving payload on `/submit`, we perform the following validation:

- Content size doesn't exceed the limit (before reading the data)
- Payload is a JSON of valid format (also check the sizes and formats of `create_at` and `block_hash`)
- `|NOW() - created_at| < 1 min`
- `submitter` is on the list `allowed` of whitelisted public keys
- `sig` is a valid signature of `data` w.r.t. `submitter` public key
- Amount of requests by `submitter` in the last hour is not exceeding `REQUESTS_PER_PK_HOURLY`

After receiving payload on `/submit` , we update in-memory public key rate-limiting state and save the contents of `block` field as `blocks/<block_hash>.dat`.

## Building

To build either a binary of the service or a Docker image, you must operate within the context of `nix-shell`. If you haven't installed it yet, follow the instructions at [install-nix](https://nix.dev/install-nix).

### Building the Binary

Enter the `nix-shell` and execute the `make` command:

```bash
$ nix-shell
[nix-shell]$ make
```

### Building the Docker Image

To create a Docker image within the `nix-shell`, set the `$TAG` environment variable for naming the image and then execute the `make docker` command:

```bash
$ nix-shell
[nix-shell]$ TAG=uptime-service-backend make docker
```

To build and publish the Docker image to GitHub Container Registry (`ghcr.io/o1-labs/uptime-service-backend`), use the [Publish](https://github.com/o1-labs/uptime-service-backend/actions/workflows/publish.yaml) GitHub Action — triggered automatically on git tags, or manually via `workflow_dispatch`.

## Testing

To run unit tests, enter the `nix-shell` and use the `make test` command:

```bash
$ nix-shell
[nix-shell]$ make test
```

To execute the integration tests, you will need the `UPTIME_SERVICE_SECRET` passphrase. This is essential to decrypt the uptime service configuration files.

### Steps to run integration tests

1. **Build the Docker Image**:

    ```bash
    $ nix-shell
    [nix-shell]$ export IMAGE_NAME=uptime-service-backend
    [nix-shell]$ export TAG=integration-test
    [nix-shell]$ make docker
    ```

2. **Run the Integration Tests**:

    ```bash
    $ nix-shell
    [nix-shell]$ export UPTIME_SERVICE_SECRET=YOUR_SECRET_HERE
    [nix-shell]$ make integration-test
    ```

> **Note:** Replace `YOUR_SECRET_HERE` with the appropriate value for `UPTIME_SERVICE_SECRET`.
