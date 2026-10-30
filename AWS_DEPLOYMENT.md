# Deploy Seatflow to AWS ECS

This guide deploys the React frontend and Go API together in one Docker container on **Amazon ECS with AWS Fargate**. The container image is stored in **Amazon ECR**, PostgreSQL runs privately on **Amazon RDS**, and an **Application Load Balancer (ALB)** provides the public application endpoint.

The steps and commands use Windows PowerShell. Replace every `<placeholder>` with your own value. Never commit actual passwords, access keys, or database URLs.

> New to AWS? Start with [AWS_ACCOUNT_SETUP.md](./AWS_ACCOUNT_SETUP.md) to create and secure an AWS account, enable MFA, and set a billing alert.

## What you need

- An AWS account with permissions to create VPC, RDS, ECR, ECS, IAM, Secrets Manager, CloudWatch Logs, and load balancer resources.
- AWS CLI configured for your account (`aws configure` or an IAM Identity Center profile).
- Docker Desktop using Linux containers.
- This project checked out locally.
- For a production HTTPS URL, a domain name you control so you can request an ACM certificate. You can use the ALB's HTTP DNS name for a temporary test, but do not use that with real user credentials.

> **Costs:** Fargate, the ALB, RDS, ECR storage, Secrets Manager, NAT gateways, and data transfer can all incur charges. A NAT gateway and an always-on load balancer/database can be significant costs. Check current AWS pricing, use the smallest suitable configuration, and delete resources when finished. Billing alerts notify you; they do not automatically stop services or cap your bill.

## Architecture

```text
Public users
    |
    v
Application Load Balancer (public subnets; HTTPS)
    |
    v
ECS Fargate service (one task in private subnets; port 8080)
    |                         |
    v                         v
Private RDS PostgreSQL     Secrets Manager
```

Keep the ECS task and RDS database private. The ALB is the only public entry point. The task runs one application process that serves both the frontend and API.

## 1. Choose a region and check your AWS account

Use the **same AWS region** for ECR, ECS, RDS, Secrets Manager, ACM, and the ALB. In the examples, replace `<region>` with your chosen region, such as `us-east-1`.

```powershell
aws sts get-caller-identity
aws configure get region
$env:AWS_DEFAULT_REGION = "<region>"
```

## 2. Create the VPC and security groups

The ALB must be able to reach ECS, and ECS must be able to reach RDS. Use security-group references instead of opening the database or application to the internet.

1. In **VPC → Your VPCs → Create VPC**, choose **VPC and more**.
2. Create a VPC with at least two Availability Zones, two public subnets, and two private subnets. Public subnets are for the ALB; private subnets are for ECS and RDS.
3. For the simplest outbound setup, choose one NAT gateway. NAT gateways incur ongoing charges. Alternatively, use the required VPC endpoints for ECR (API and Docker), S3, Secrets Manager, and CloudWatch Logs, and ensure the private subnets have routes to those endpoints.
4. Create three security groups in this VPC:

   | Security group | Inbound rules | Outbound |
   | --- | --- | --- |
   | `seatflow-alb` | TCP 443 from `0.0.0.0/0` (and `::/0` if enabling IPv6), plus TCP 80 if redirecting HTTP to HTTPS. For a temporary HTTP-only test, allow TCP 80 instead. | TCP 8080 to `seatflow-ecs`. |
   | `seatflow-ecs` | TCP 8080 **from `seatflow-alb` only**. | TCP 5432 to `seatflow-db`, plus HTTPS to the configured NAT/endpoints for AWS services. |
   | `seatflow-db` | PostgreSQL TCP 5432 **from `seatflow-ecs` only**. | Default outbound is sufficient. |

Do not allow port 5432 from a public IP range. Do not assign public IP addresses to the ECS tasks.

## 3. Create the private PostgreSQL database

In **Amazon RDS → Databases → Create database**:

1. Choose **Standard create**, PostgreSQL, and a currently supported version.
2. Select a small development-appropriate database configuration.
3. Set the initial database name to `seatflow`. Choose a database master username and a **new, strong password**. Do not reuse a password posted in chat or used for pgAdmin.
4. Enable managed credentials in Secrets Manager if offered, or create the connection URL secret in the next section.
5. Select the VPC created above. Set **Public access** to **No**.
6. Create or select a DB subnet group using the VPC's private subnets.
7. Attach the `seatflow-db` security group.
8. Create the database and wait for status **Available**. Copy its endpoint hostname from **Connectivity & security**.

The Go application applies `server/schema.sql` at startup. The configured database user must be allowed to create and update the application tables, indexes, and seed rows. A dedicated application/migration role with limited permissions is preferable to using the RDS master user for a production deployment.

### Store the database URL in Secrets Manager

In **AWS Secrets Manager → Store a new secret → Other type of secret**, store the complete connection URL as a plain-text secret named `seatflow/production/database-url`:

```text
postgres://<database-user>:<URL-encoded-password>@<rds-endpoint>:5432/seatflow?sslmode=require
```

An RDS endpoint usually resembles `seatflow.abc123.<region>.rds.amazonaws.com`. Percent-encode URL-reserved characters in the password, such as `@`, `:`, `/`, `#`, or `%`. Never put the real URL in source code, this guide, or a frontend variable.

`sslmode=require` encrypts the database connection. For certificate and hostname verification, configure the AWS RDS CA certificate in the container and use `sslmode=verify-full`.

## 4. Create an ACM certificate for HTTPS (recommended)

An ALB HTTPS listener needs a certificate for a domain you control; you cannot issue a certificate for the ALB's generated AWS DNS name.

1. Open **AWS Certificate Manager (ACM)** in the same region as the ALB.
2. Request a public certificate for a domain or subdomain you control, such as `booking.example.com`.
3. Complete DNS validation by adding the ACM-provided CNAME record to your domain's DNS.
4. Wait until the certificate status is **Issued**.

You will point the domain to the ALB after creating it. If you do not have a domain yet, you can configure a temporary HTTP listener for testing. Use `COOKIE_SECURE=false` only for that temporary test, do not enter real credentials, and switch to HTTPS with `COOKIE_SECURE=true` before real use.

## 5. Create an ECR repository and push the image

In **Amazon ECR → Private registry → Repositories → Create repository**, create a repository named `seatflow`. Copy its URI, which resembles `<account-id>.dkr.ecr.<region>.amazonaws.com/seatflow`.

From the project root in PowerShell, authenticate, build, and push the Docker image:

```powershell
$accountId = "<account-id>"
$region = "<region>"
$repository = "$accountId.dkr.ecr.$region.amazonaws.com/seatflow"
$tag = "release-1"

aws ecr get-login-password --region $region |
  docker login --username AWS --password-stdin "$accountId.dkr.ecr.$region.amazonaws.com"
if ($LASTEXITCODE -ne 0) { throw "ECR login failed." }

docker build -t "seatflow:$tag" .
if ($LASTEXITCODE -ne 0) { throw "Docker image build failed." }

docker tag "seatflow:$tag" "$repository`:$tag"
docker push "$repository`:$tag"
if ($LASTEXITCODE -ne 0) { throw "Docker image push failed." }
```

The Docker build compiles the React app and Go API and packages them in one production image. Use a new image tag for every release so deployments and rollbacks are identifiable.

## 6. Create the ECS cluster and task definition

### Create the cluster

1. Open **Amazon ECS → Clusters → Create cluster**.
2. Name it `seatflow`.
3. Choose the AWS Fargate/serverless infrastructure option if shown, then create the cluster.

### Create IAM roles

The task definition uses two different roles:

- **Task execution role:** lets ECS pull the image from ECR, write container logs to CloudWatch, and read the database URL secret. Create or select a role with the AWS-managed `AmazonECSTaskExecutionRolePolicy` policy, then add `secretsmanager:GetSecretValue` permission scoped to the specific secret ARN. If the secret uses a customer-managed KMS key, also grant `kms:Decrypt` for that key.
- **Task role:** permissions for AWS API calls made by the application inside the container. Seatflow does not call AWS APIs from the application, so no additional task-role permissions are needed.

Do not give either role administrator access. The secret permission for environment-variable injection belongs on the **execution role**.

### Register a task definition

In **Amazon ECS → Task definitions → Create new task definition**:

1. Choose **AWS Fargate** as the launch type, Linux/x86_64 as the operating system/architecture, and an appropriate small task CPU/memory size.
2. Set the task execution role created above.
3. Add a container named `seatflow` with the ECR image URI including the release tag.
4. Set the container port to `8080` using TCP.
5. Configure the container's log driver as `awslogs` and choose a CloudWatch log group (create one if prompted).
6. Add these environment variables:

   | Name | Value |
   | --- | --- |
   | `PORT` | `8080` |
   | `COOKIE_SECURE` | `true` when using HTTPS; temporary HTTP test only: `false` |
   | `WEB_DIST` | `/app/dist` |
   | `FRONTEND_ORIGIN` | The final browser origin, e.g. `https://booking.example.com` (no trailing slash). For a temporary HTTP test, use `http://<alb-dns-name>`. |

7. Add `DATABASE_URL` under the container's **Secrets** section (not Environment variables). Select the Secrets Manager secret `seatflow/production/database-url`.
8. Register the task definition.

The ECS service injects the secret when a task starts. After rotating the secret, deploy a new task (for example, force a new deployment) for the application to use the updated value.

## 7. Create the ECS service and public ALB

In **Amazon ECS → Clusters → seatflow → Services → Create**:

1. Choose the Fargate launch type and the task definition/revision registered above.
2. Name the service `seatflow-service` and set the desired task count to **1**.
3. In networking, select the VPC created above and its **private subnets**. Disable **Auto-assign public IP**. Attach the `seatflow-ecs` security group.
4. Enable a load balancer and choose **Application Load Balancer**. Create a new ALB named `seatflow-alb` in the **public subnets**, with the `seatflow-alb` security group.
5. Create a target group for the ECS service using **IP** target type and HTTP port `8080`. Set the health check protocol to HTTP and path to `/api/healthz`.
6. For production HTTPS, configure an HTTPS listener on port `443` and select the ACM certificate from step 4. Configure HTTP port `80` to redirect to HTTPS. For a temporary test without a domain/certificate, use an HTTP listener on port `80`.
7. Set deployment circuit breaker/rollback if the console offers it. Keep the service at one task: Seatflow's live-update broadcaster is process-local, and multiple tasks do not share SSE notifications. Scaling beyond one task requires shared pub/sub such as PostgreSQL `LISTEN/NOTIFY`.
8. Create the service and wait until the task is **Running** and the target group reports it as **Healthy**.

The health endpoint checks database connectivity, so the task must be able to reach RDS before the ALB health check succeeds. The task security group must allow port 5432 to `seatflow-db`, and the DB security group must allow port 5432 from `seatflow-ecs`.

### Point your domain to the ALB

After the ALB is active, copy its DNS name from **EC2 → Load Balancers**. In the DNS provider for your domain, create an alias/CNAME record pointing the chosen hostname (for example `booking.example.com`) to the ALB DNS name. Wait for DNS and the ACM certificate to validate.

Set `FRONTEND_ORIGIN` in the task definition to the exact HTTPS origin, then update the ECS service to deploy a new task. For example:

```text
https://booking.example.com
```

The application URL is your domain. The ALB DNS name alone is useful for temporary HTTP testing, but does not support a trusted HTTPS certificate unless you control a matching domain.

## 8. Verify the deployment

Open the application and health check, replacing the example hostname with your domain:

- Application: `https://booking.example.com/`
- Health: `https://booking.example.com/api/healthz`

The health endpoint should return:

```json
{"status":"ok"}
```

Then test the application end to end:

1. Create an account from **Profile** or **Join us**.
2. Open an event, select seats, and confirm the booking.
3. Verify the seats appear under **My bookings**.
4. Cancel the booking and confirm the seats become available again.

## 9. Troubleshoot common problems

### ECS task stops or the target is unhealthy

- Check **ECS → Cluster → Service → Tasks** for the stopped reason.
- Check the container's CloudWatch log stream for startup errors.
- Confirm the task definition uses port `8080`, and the target group uses HTTP port `8080` with health path `/api/healthz`.
- Verify that the task is in private subnets and has outbound access to ECR, Secrets Manager, and CloudWatch Logs through a NAT gateway or the required VPC endpoints.
- Confirm the execution role can read the specific database secret.
- Verify the ECS security group allows outbound PostgreSQL traffic to the DB security group, and the DB security group allows inbound port 5432 from the ECS security group.
- Check the task's `DATABASE_URL`, RDS endpoint, database name `seatflow`, and database permissions. The API initializes `schema.sql` during startup.

### `password authentication failed` or database connection errors

- Check the username/password in Secrets Manager; these are the RDS credentials, not your local pgAdmin login unless you intentionally set them to match.
- URL-encode special characters in the password in the connection URL.
- Confirm RDS is **Available**, RDS and ECS use the same VPC, and the database remains private.
- Confirm the URL ends with `/seatflow?sslmode=require`.

### Browser, cookie, or CORS problems

- Use the HTTPS domain and set `COOKIE_SECURE=true` for a real deployment.
- Set `FRONTEND_ORIGIN` to the exact browser origin, including scheme and hostname but no path or trailing slash.
- Ensure DNS points to the ALB and the ACM certificate covers that domain.
- For a temporary HTTP test only, set `COOKIE_SECURE=false` and use the exact `http://<alb-dns-name>` origin. Do not use real credentials over HTTP.

## 10. Update or remove the deployment

To publish an update, build and push a new ECR image tag, register a new task-definition revision with that image, then update the ECS service to use the revision. Monitor ECS events, task logs, and ALB target health during deployment.

To stop incurring charges, first export or retain any database data you need. Then delete the ECS service and cluster, ALB and target group, RDS database, unneeded ECR images/repository, Secrets Manager secret, CloudWatch log group, and networking resources created only for Seatflow (including NAT gateways). Deleting RDS is destructive; take a snapshot if you need the data.

## Important notes

- This guide describes a deployment you create in your AWS account; no AWS resources or public URL have been created by this repository.
- RDS's endpoint is private database connectivity, not the public application URL.
- The application initializes its schema during startup for convenience. For larger production deployments, move schema changes to a versioned migration job.
- Run one ECS task for the current process-local SSE hub. Use shared pub/sub before increasing the task count.
- Keep production credentials in Secrets Manager and rotate them periodically.
