# URL Shortener API

This is a simple URL shortener API built with Go. It provides endpoints to create, read, update, delete and redirect shortened URLs.

This project leverages a robust tech stack including [Go](https://github.com/golang/go) for the core implementation, [Gin](https://github.com/gin-gonic/gin) as the web framework, [validator](https://github.com/go-playground/validator) for input validation, [rate](https://pkg.go.dev/golang.org/x/time/rate) for rate limiting, [logrus](https://github.com/sirupsen/logrus) for logging, and [testify](https://github.com/stretchr/testify) for testing (assertions/mocks). It's containerized with [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) and performance-tested using [k6](https://k6.io/).

## Getting Started

### Prerequisites

- [Go 1.22 or higher](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) (for containerized deployment)

### Local Development

1. Clone the repository.

2. Install dependencies:
   ```
   make deps
   # or
   go mod download
   ```

3. Build the project:
   ```
   make build
   # or
   go build -o urlshortener
   ```

4. Run the server:
   ```
   make run
   # or
   go run main.go
   # or after building
   ./urlshortener
   ```

The server will start on `http://localhost:3000` by default.

### Available Commands

This project supports both `make` and `go` commands for various tasks. For example:

- Building: `make build` or `go build`
- Testing: `make test` or `go test ./...`
- Integration Testing: `make integration-test` or `go test -tags=integration ./tests/integration/...`
- Running: `make run` or `go run main.go`
- Coverage: `make coverage` or `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html`
- Cleaning: `make clean` or manually remove build artifacts

For a full list of available `make` commands, run `make help`.

### Docker Deployment

   To run the application:
   ```sh
   docker-compose up --build app
   ```

## API Documentation

The API is documented using OpenAPI (Swagger) specification. To view the interactive documentation:

1. Copy the contents of [openapi.yml](openapi.yaml)
2. Visit [Swagger Editor](https://editor.swagger.io/)
3. Paste the contents into the editor

This provides an interactive interface to explore the API endpoints, request/response formats, and even test the API directly.

## API Endpoints

- `POST /api/v1/short`: Create a short URL
- `GET /api/v1/short/:short_url`: Get URL data
- `PUT /api/v1/short/:short_url`: Update a short URL
- `DELETE /api/v1/short/:short_url`: Delete a short URL
- `GET /health`: Health check
- `GET /:short_url`: Redirect to original URL

## Performance Testing

Run k6 performance tests:

- With Docker Compose:
  ```
  docker-compose run k6
  ```
  
## Configuration

Key configuration options (found in `config/config.go`):

- `RateLimit`: Requests per second limit (default: 10)
- `ServerPort`: Server listening port (default: 3000)
- `RequestTimeout`: Timeout for API requests
- `DisableRateLimit`: Used for local development and running performance tests (default: false)

## Continuous Integration

This project uses GitHub Actions for Continuous Integration (CI). The CI workflow is defined in the `.github/workflows/ci.yml` file. It includes steps for:

### CI Workflow
The CI workflow is defined in the `.github/workflows/ci.yml` file. It includes steps for:
- Checking out the code
- Setting up Go
- Installing dependencies
- Building the project
- Running unit tests
- Running integration tests
- Building the Docker image
- Archiving test results

The CI pipeline is triggered on pushes and pull requests to the `main` branch.

### Performance Testing Workflow
The performance testing workflow is defined in the `.github/workflows/k6.yml` file. It includes steps for:
- Checking out the code
- Running k6 performance tests

The performance tests can be manually triggered using the `workflow_dispatch` event. This allows you to run performance tests against any branch manually from the GitHub Actions tab.

## Example Usage with CURL

Here are some examples of how to use the API with CURL:

1. Create a short URL:
   [Create Short URL](http://localhost:3000/api/v1/short) (POST)
   ```sh
   curl -X POST -H "Content-Type: application/json" -d '{"url":"https://www.example.com/very/long/url"}' http://localhost:3000/api/v1/short
   ```

2. Get URL data:
   [Get URL Data](http://localhost:3000/api/v1/short/abc123) (GET)
   ```sh
   curl http://localhost:3000/api/v1/short/abc123
   ```

3. Update a short URL:
   [Update Short URL](http://localhost:3000/api/v1/short/abc123) (PUT)
   ```sh
   curl -X PUT -H "Content-Type: application/json" -d '{"url":"https://www.example.com/updated/url"}' http://localhost:3000/api/v1/short/abc123
   ```

4. Delete a short URL:
   [Delete Short URL](http://localhost:3000/api/v1/short/abc123) (DELETE)
   ```sh
   curl -X DELETE http://localhost:3000/api/v1/short/abc123
   ```

5. Health check:
   [Health Check](http://localhost:3000/health) (GET)
   ```sh
   curl http://localhost:3000/health
   ```

6. Redirect to original URL:
   [Redirect to Original URL](http://localhost:3000/abc123) (GET)
   ```sh
   curl -L http://localhost:3000/abc123
   ```

Replace `abc123` with an actual short URL generated by the service.
## Git Hooks

This project uses git hooks to maintain code quality and consistency. I have included a pre-push hook that runs unit tests before each push, ensuring that only code passing all tests is pushed to the repository.

To set up the git hooks:

1. After cloning the repository, run:
   ```sh
   sh setup-hooks.sh
   ```

2. The script will install the pre-push hook in your local `.git/hooks` directory.

As a developer, you'll benefit from this setup in several ways:
- It acts as a safety net, preventing you from accidentally pushing code that breaks tests.
- It encourages writing and maintaining tests as an integral part of the development process.
- It helps maintain the overall quality and stability of the codebase.

If you need to bypass the hook for any reason (not recommended for regular use), you can use the `--no-verify` flag with your git push command:

```sh
git push --no-verify
```

## Future Improvements

- Implement persistent storage (e.g., database integration)
- Add comprehensive metrics and monitoring
- Implement user authentication and URL ownership
- Run distributed load tests on a kubernetes cluster
