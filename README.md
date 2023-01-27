# Refinery Rule Updater

This will give you a local UI for updating your Refinery rules in s3 or a local file.

## Run

`go build`

`./crude`

If it starts up, visit [http://localhost:8000]()

## Config

set up environment variables:

```
export HONEYCOMB_API_KEY="your-api-key-here"
export OTEL_SERVICE_NAME="crude"
export S3_BUCKET=the-s3-bucket
```

configure your AWS connection

`aws configure` (probably)

## S3 bucket

The rules will be read from a bucket $S3_BUCKET

TODO: example

## Local file

The local file will be read and written to `/tmp/rules.txt` (in TOML format?)
