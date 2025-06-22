provider "aws" {
  region = "us-west-2"
}

provider "google" {
  project = var.project_id
  region  = var.region
}
