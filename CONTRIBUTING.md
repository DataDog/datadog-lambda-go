# Contributing

We love pull requests. Here's a quick guide.

1. Fork, clone and branch off:
    ```bash
    git clone git@github.com:<your-username>/datadog-lambda-go.git
    git checkout -b <my-branch>
    ```
1. Make your changes, update tests and ensure the tests pass:
    ```bash
    go test ./...
    ```
1. Build and test your own serverless application with your modified version of `datadog-lambda-go`.
1. Push to your fork and [submit a pull request][pr].

[pr]: https://github.com/your-username/datadog-lambda-go/compare/DataDog:main...main

At this point you're waiting on us. We may suggest some changes or improvements or alternatives.
