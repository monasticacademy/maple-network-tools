# Not a secret...
SLACK_APP_ID := A03QG3JGNDQ
SLACK_CLIENT_ID := 2480523769.3832120566466

PROJECT := thankyou-letters
REGION := us-east4
TOPIC := print-queue

# This one must match the function name in golang
FUNCTION := print-from-slack
ENTRYPOINT := HandleSlackEvent
ENDPOINT := https://$(REGION)-$(PROJECT).cloudfunctions.net/$(FUNCTION)

# The service account that we terraform uses
SERVICE_ACCOUNT := terraform
SERVICE_ACCOUNT_KEY := secrets/terraform_service_account.json
SERVICE_ACCOUNT_EMAIL := $(SERVICE_ACCOUNT)@$(PROJECT).iam.gserviceaccount.com

# The service account that the cloud function uses at runtime
RUNTIME_SERVICE_ACCOUNT := $(PROJECT)@appspot.gserviceaccount.com

# Run in "development" mode
run:
	env \
		GOOGLE_CLOUD_PROJECT=$(PROJECT) \
		TOPIC=$(TOPIC) \
	go run ./harness

deploy:
	gcloud functions deploy $(FUNCTION) \
		--entry-point $(ENTRYPOINT) \
		--region $(REGION) \
		--trigger-http \
		--allow-unauthenticated \
		--runtime=go116 \
		--set-env-vars=TOPIC=$(TOPIC) \
		--set-secrets=SLACK_TOKEN=slack-client-secret:latest,SLACK_SIGNING_SECRET=slack-signing-secret:latest

open-runtime-logs:
	firefox https://console.cloud.google.com/functions/details/$(REGION)/$(FUNCTION)?env=gen1&project=$(PROJECT)&tab=logs

open-build-logs:
	firefox https://console.cloud.google.com/cloud-build/builds;region=$(REGION)/35e814bb-e9bf-49f9-81b4-9b2f1bf7276e?project=801997860052

create-service-account:
	gcloud iam service-accounts create $(SERVICE_ACCOUNT)
	gcloud iam service-accounts keys create $(SERVICE_ACCOUNT_KEY) \
		--iam-account $(SERVICE_ACCOUNT_EMAIL)
	gcloud projects add-iam-policy-binding $(PROJECT) \
		--member serviceAccount:$(SERVICE_ACCOUNT_EMAIL) \
		--role roles/editor

grant-appengine-access-to-secrets:
	gcloud projects add-iam-policy-binding $(PROJECT)
		--member serviceAccount:$(RUNTIME_SERVICE_ACCOUNT) \
		--role roles/secretmanager.secretAccessor

encrypt-secrets:
	@echo "Please enter the password from bitwarden under 'MAPLE website secrets'..."
	@crypt --encrypt $(shell ls secrets/* | grep -v encrypted$$)

decrypt-secrets:
	@echo "Please enter the password from bitwarden under 'MAPLE website secrets'..."
	@crypt --decrypt secrets/*.encrypted
