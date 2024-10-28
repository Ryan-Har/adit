terraform {
  backend "s3" {
    bucket         = "rharris.dev.terraform.state"
    key            = "adit/terraform/state"
    region         = "eu-west-2"
    dynamodb_table = "rharris.dev.terraform.lock"
    encrypt        = true
  }
}

provider "aws" {
  region = "eu-west-2"
}