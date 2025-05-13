# Fingrab
[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/HallyG/fingrab/master.yaml)](https://github.com/HallyG/fingrab/actions/workflows/master.yaml)
[![GitHub Release](https://img.shields.io/github/v/release/hallyg/fingrab?label=latest%20release)](https://github.com/hallyg/fingrab/releases/latest)
[![License](https://img.shields.io/github/license/hallyg/fingrab)](https://github.com/HallyG/fingrab/blob/master/LICENSE)
![Go Version](https://img.shields.io/github/go-mod/go-version/hallyg/fingrab)

A CLI for exporting financial data from various banks.

Currently supports: [Monzo](https://monzo.com/), and [Starling](https://www.starlingbank.com/).

## Table of Contents
- [Disclaimer](#disclaimer)
- [Installation](#installation)
  - [Install via Go](#install-via-go)
  - [From Source](#from-source)
  - [Using Docker](#using-docker)
- [Usage](#usage)
  - [Obtaining API Tokens](#obtaining-api-tokens)
    - [Monzo](#monzo)
    - [Starling](#starling)
  - [Exporting Transactions](#exporting-transactions)
    - [Monzo](#monzo-1)
    - [Starling](#starling-1)
- [Contributing](#contributing)
  - [New Format](#new-format)
- [License](#license)

## Disclaimer
This project is **not affiliated with or endorsed by** Starling Bank or Monzo. It is an unofficial, independent, open-source implementation using their public APIs.

## Installation

### Install via Go
1. Ensure you have [Go 1.24](https://go.dev/doc/install) or later installed.
2. Install the clu:
   ```bash
   go install github.com/HallyG/fingrab@latest
   ```

### From Source
1. Ensure you have [Go 1.24](https://go.dev/doc/install) or later installed.
2. Clone the repository:
   ```bash
   git clone https://github.com/hallyg/fingrab.git
   cd fingrab
   ```
3. Build the project:
   ```bash
   make build
   ```

### Using Docker
1. Ensure you have [Docker](https://docs.docker.com/get-started/get-docker/) installed.
2. Build the docker image:
   ```bash
   make docker/build
   ```

## Usage

### Obtaining API Tokens

#### Monzo
1. Go to [Monzo Developer Portal](https://developers.monzo.com/).
2. Log in to your Monzo developer account.
3. Open the Monzo App and allow API Playground access to your account.
4. Use the displayed access token.

#### Starling
1. Go to [Starling Developer Portal](https://developer.starlingbank.com/).
2. Log in to your Starling developer account.
3. Create a new [personal access token](https://developer.starlingbank.com/personal/token).
4. Use the generated access token.

### Exporting Transactions

#### Monzo
```bash
# API Auth with cli flag
fingrab export monzo --token <monzo-api-token> --start 2025-03-01 --end 2025-03-31

# API Auth with env var
export MONZO_TOKEN=<monzo-api-token>
fingrab export monzo --start 2025-03-01 --end 2025-03-31

# Exporting to Moneydance format
fingrab export monzo --token <monzo-api-token> --start 2025-03-01 --end 2025-03-31 --format moneydance

# Verbose output
fingrab export monzo --token <monzo-api-token> --start 2025-03-01 --end 2025-03-31 --verbose
```

#### Starling
```bash
# API Auth with cli flag
fingrab export starling --token <starling-api-token> --start 2025-03-01 --end 2025-03-31

# API Auth with env var
export STARLING_TOKEN=<starling-api-token>
fingrab export starling --start 2025-03-01 --end 2025-03-31

# Exporting to Moneydance format
fingrab export starling --token <starling-api-token> --start 2025-03-01 --end 2025-03-31 --format moneydance

# Verbose output
fingrab export starling --token <starling-api-token> --start 2025-03-01 --end 2025-03-31 --verbose
```
## Contributing

### New Format
To add a new format for exporting financial data, follow these steps:
1. Navigate to the `internal/format`.
2. Create a new Go file for your format (e.g. `exampleformat.go`).
3. Add a new format by implementing the `Formatter` interface defined in `format.go`. For example:
```go
package format

import (
	"io"

	"github.com/HallyG/fingrab/internal/domain"
)

const FormatTypeExample FormatType = "newformat"

type ExampleFormatter struct {
	w io.Writer
}

func NewExampleFormatter(w io.Writer) *ExampleFormatter {
	return &ExampleFormatter{w: w}
}

func (e *ExampleFormatter) WriteHeader() error {
	_, err := e.w.Write([]byte("header content\n"))
	return err
}

func (e *ExampleFormatter) WriteTransaction(transaction *domain.Transaction) error {
	_, err := e.w.Write([]byte("transaction content\n"))
	return err
}

func (e *ExampleFormatter) Flush() error {
	return nil
}

func init() {
	Register(FormatTypeExample, func(w io.Writer) Formatter {
		return NewExampleFormatter(w)
	})
}
```
4. Ensure the init function registers the new format with a unique `FormatType`.

## License
This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.