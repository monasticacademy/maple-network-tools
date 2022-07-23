variable "project_id" {
    type = string
    default = "thankyou-letters"
}

provider "google" {
  credentials = file("secrets/terraform_service_account.json")
  project     = var.project_id
}

resource "google_secret_manager_secret" "slack-client-secret" {
    secret_id = "slack-client-secret"
    replication {
        automatic = true
    }
}

resource "google_secret_manager_secret_version" "slack-client-secret" {
    secret = resource.google_secret_manager_secret.slack-client-secret.id
    secret_data = file("${path.module}/secrets/slack-client-secret")
}

resource "google_secret_manager_secret" "slack-signing-secret" {
    secret_id = "slack-signing-secret"
    replication {
        automatic = true
    }
}

resource "google_secret_manager_secret_version" "slack-signing-secret" {
    secret = resource.google_secret_manager_secret.slack-signing-secret.id
    secret_data = file("${path.module}/secrets/slack-signing-secret")
}

resource "google_secret_manager_secret" "slack-token" {
    secret_id = "slack-token"
    replication {
        automatic = true
    }
}

resource "google_secret_manager_secret_version" "slack-token" {
    secret = resource.google_secret_manager_secret.slack-token.id
    secret_data = file("${path.module}/secrets/slack-token")
}
