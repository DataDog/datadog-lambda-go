# Hello Dog

Used for testing the go-lambda-dd.

## Requirements

- Node
- Go
- Dep
- Make

## Building

```bash
make clean
make build
```

## Deployment

To deploy to the demo account use the following

```bash
aws-vault exec demo-account-admin -- sls deploy --api-key <API_KEY>
```
