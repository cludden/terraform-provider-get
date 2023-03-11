# terraform-provider-get
a [Terraform](https://www.terraform.io/) provider for downloading remote artifacts via [go-getter](https://github.com/hashicorp/go-getter)

## Getting Started

```terraform
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 3.0.0"
    }
    get = {
      source  = "cludden/get"
      version = "0.1.2"
    }
  }
}

# download lambda artifact
resource "get_artifact" "benthos_serverless" {
  url      = "https://github.com/Jeffail/benthos/releases/download/v3.62.0/benthos-lambda_3.62.0_linux_amd64.zip"
  checksum = "file:https://github.com/Jeffail/benthos/releases/download/v3.62.0/benthos_3.62.0_checksums.txt"
  dest     = "benthos-lambda_3.62.0_linux_amd64.zip"
  mode     = "file"
  archive  = false
  workdir  = abspath(path.root)
}

# provision lambda function
resource "aws_lambda_function" "this" {
  filename         = get_artifact.benthos_serverless.dest
  function_name    = var.name
  handler          = "benthos-lambda"
  role             = var.role_arn
  runtime          = "go1.x"
  source_code_hash = get_artifact.benthos_serverless.sum64
  timeout          = 30

  environment {
    variables = {
      BENTHOS_CONFIG = <<-YAML
        output:
          broker:
            pattern: fan_out
            outputs:
            - kafka:
                addresses:
                - todo:9092
                client_id: benthos_serverless
                topic: example_topic
            - sync_response: {}
      YAML
    }
  }
}
```

## License
Licensed under the [MIT License](LICENSE.md)  
Copyright (c) 2022 Chris Ludden