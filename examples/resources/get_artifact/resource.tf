# download github archive
resource "get_artifact" "example" {
  url      = "https://github.com/Jeffail/benthos/releases/download/v3.62.0/benthos-lambda_3.62.0_linux_amd64.zip"
  checksum = "file:https://github.com/Jeffail/benthos/releases/download/v3.62.0/benthos_3.62.0_checksums.txt"
  dest     = "benthos-lambda_3.62.0_linux_amd64.zip"
  mode     = "file"
  archive  = false
  workdir  = abspath(path.root)
}
