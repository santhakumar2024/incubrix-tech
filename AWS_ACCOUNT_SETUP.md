# AWS account setup for beginners

This guide helps you create and secure your own AWS account before deploying Seatflow. You must complete signup yourself; AWS will ask you to verify your email and phone and provide a payment method.

> **Billing warning:** An AWS account is not a guarantee that hosting will be free. ECS Fargate, RDS, load balancers, networking, and other resources can incur charges. Read the current pricing and offers shown during signup, create a billing alert, and remove resources when you are finished. A billing alert notifies you; it does not automatically stop services or cap your bill.

## What to have ready

- An email address you control and can access.
- A unique, strong password. Do not reuse your email or pgAdmin password.
- Your name, phone number, and address.
- A valid payment method accepted by AWS in your country.
- An authenticator app on your phone for multi-factor authentication (MFA).

Do not send your password, card details, verification codes, AWS access keys, or database credentials to anyone, including in chat.

## 1. Create your AWS account

1. Open the official [AWS account signup page](https://portal.aws.amazon.com/billing/signup).
2. Select **Create a new AWS account**.
3. Enter your email address and a clear account name, such as `Seatflow Learning`. The email and password used for the **root user** give full control over the account.
4. Verify the email address with the code AWS sends you.
5. Create a strong, unique root-user password and store it somewhere safe, such as a trusted password manager.
6. Select the account type that describes you (for a personal learning project, this is usually **Personal**), and enter your contact details accurately.
7. Enter a payment method and complete any identity or phone verification AWS requests. AWS may place a temporary authorization on the payment method; follow the current explanation shown by AWS.
8. Choose a support plan. Review the price displayed and choose the no-cost/basic option if it is available and appropriate for you; paid support is not required just to follow this guide.
9. Wait for AWS to confirm that the account is ready. This can take a little while.
10. Open the [AWS Management Console](https://console.aws.amazon.com/). Sign in using **Root user** and the email and password you just registered.

AWS’s [account setup guide](https://docs.aws.amazon.com/accounts/latest/reference/manage-acct-creating.html) explains the signup process. Offers and signup screens can vary by country and change over time, so use the current AWS screen rather than assuming a particular free offer.

## 2. Secure the root user

The root user has full access to the AWS account. Secure it before creating any hosting resources:

1. In the AWS Console, select your account name, then **Security credentials** (or search for **IAM** and open the root-user security settings).
2. Find **Multi-factor authentication (MFA)** and choose **Assign MFA device**.
3. Choose an authenticator app (virtual MFA), follow AWS’s QR-code setup, and enter the requested one-time codes.
4. Save any recovery information AWS provides somewhere safe. Do not share it.
5. Use the root-user login only for account-owner tasks. Do not create root access keys or use the root login for everyday AWS CLI work.

See AWS’s [root-user security recommendations](https://docs.aws.amazon.com/IAM/latest/UserGuide/root-user-best-practices.html). Also secure the email account you used to register, since it is part of your AWS account recovery.

## 3. Add a billing alert before deploying

1. In the Console’s search box, find and open **Billing and Cost Management**.
2. Open **Budgets** and choose **Create budget**.
3. Choose a **cost budget** and set a small monthly amount you would notice (for example, `$10`—pick an amount that makes sense for you).
4. Add your email address for notifications and configure alerts when actual or forecasted cost reaches thresholds such as 50%, 80%, and 100%.
5. Create the budget and confirm any email verification AWS requires.

This budget sends alerts; it **does not** stop or delete AWS resources when the amount is reached. Check the [AWS Budgets guide](https://docs.aws.amazon.com/cost-management/latest/userguide/budgets-create.html).

## 4. Know the two different logins

- **AWS root login:** the signup email and password. It controls the account. Protect it with MFA and avoid using it for routine work.
- **PostgreSQL login:** a database username and password used by Seatflow to connect to RDS. It is separate from the AWS login and from your local pgAdmin login.

You will create or choose a separate PostgreSQL password during the RDS setup. Store the database connection URL in **AWS Secrets Manager**, as described in [AWS_DEPLOYMENT.md](./AWS_DEPLOYMENT.md). Do not put database credentials in frontend code, this guide, or a public Git repository.

## 5. Pick your AWS region

In the Console, choose a region from the region selector in the top navigation. Use the same region for RDS, Secrets Manager, ECR, ECS, and the Application Load Balancer. Pick a region close to you where those services are available, and note its exact code (for example, `us-east-1`). AWS services and prices vary by region.

## 6. Continue with Seatflow deployment

Once signup is complete, root MFA is enabled, and a billing alert is set, continue at [AWS_DEPLOYMENT.md](./AWS_DEPLOYMENT.md). That guide walks through creating PostgreSQL on RDS, protecting its password in Secrets Manager, building and publishing the Docker image to ECR, and running the application on ECS Fargate behind a public Application Load Balancer.

**Pause before creating RDS, ECS, a load balancer, or a NAT gateway:** these resources may start incurring charges. Review their current prices and the cleanup steps in the deployment guide first.
