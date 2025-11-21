=============================
terraform-provider-jsonschema
=============================

.. image:: https://codecov.io/github/iilei/terraform-provider-jsonschema/branch/master/graph/badge.svg
    :target: https://codecov.io/github/iilei/terraform-provider-jsonschema
    :alt: Coverage Status

A |terraform|_ provider for validating JSON, JSON5, YAML, and TOML documents using |json-schema|_ specifications.

.. warning::
   ⚠️ **Breaking Changes in v0.6.0**

   Version 0.6.0 introduces **breaking API changes**:

   - **document field**: Now expects a file path instead of content (remove ``file()`` wrapper)
   - **valid_json field**: Renamed from ``validated`` for clarity
   - **force_filetype field**: New optional field to override format detection
   - **Multi-format support**: Added YAML and TOML validation

   **Migration required:**

   .. code-block:: diff

     # Before (v0.5.x)
     data "jsonschema_validator" "config" {
     -  document = file("${path.module}/config.json")
     +  document = "${path.module}/config.json"
        schema   = "${path.module}/config.schema.json"
     }

     locals {
     -  config = jsondecode(data.jsonschema_validator.config.validated)
     +  config = jsondecode(data.jsonschema_validator.config.valid_json)
     }

   See the full documentation for migration details.

.. warning::
   ⚠️ **Version 0.x Development - Breaking Changes Expected**

   This provider is in initial development (0.x.x). Per `semantic versioning <https://semver.org/#spec-item-4>`_, **breaking changes may occur in ANY release** (minor or patch) until version 1.0.0.

   **Required actions:**

   - **Always pin to a specific version** in production
   - **Review release notes** before upgrading
   - **Test upgrades** in non-production environments first

   Stability and standard semver guarantees begin at version 1.0.0.

Features
========

- **Multi-format Support**: Validate JSON, JSON5, YAML, and TOML documents against JSON Schema
- **Auto-detection**: Format determined from file extension (``.json``, ``.json5``, ``.yaml``, ``.yml``, ``.toml``)
- **JSON5 Support**: Full support for JSON5 schemas with comments, trailing commas, unquoted keys
- **Schema Versions**: Draft 4, 6, 7, 2019-09, and 2020-12 support
- **External References**: Resolves ``$ref`` URIs including JSON5 files
- **Reference Overrides**: Redirect remote ``$ref`` URLs to local files for offline validation
- **Enhanced Error Templating**: Flexible error formatting with Go templates
- **Deterministic Output**: Consistent JSON for stable Terraform state

See the `full documentation <docs/index.md>`_ for detailed usage, advanced features, and examples. **Hands-on examples** demonstrating schema traversal, error templating, and reference overrides are available in the `examples/ <examples/>`_ directory.

Installation
============

Terraform Provider
------------------

On |terraform|_ versions 0.13+ use:

.. code-block:: terraform

  terraform {
    required_providers {
      jsonschema = {
        source  = "iilei/jsonschema"
        version = "0.6.0"  # Pin to specific version
      }
    }
  }

For |terraform|_ versions 0.12 or lower, see |terraform-install-plugin|_.

Standalone CLI Tool
-------------------

Install the ``jsonschema-validator`` CLI for use outside Terraform (Python, Node.js, or any project):

**Via Go:**

.. code-block:: bash

  go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest

**Via Release Binary:**

Download pre-built binaries from `GitHub Releases <https://github.com/iilei/terraform-provider-jsonschema/releases>`_


Quick Start
===========

Terraform Provider
------------------

.. code-block:: terraform

  provider "jsonschema" {
    schema_version = "draft/2020-12"  # Optional
  }

  # JSON validation
  data "jsonschema_validator" "config" {
    document = "${path.module}/config.json"
    schema   = "${path.module}/config.schema.json"
  }

  # YAML validation (auto-detected from .yaml extension)
  data "jsonschema_validator" "k8s_manifest" {
    document = "${path.module}/deployment.yaml"
    schema   = "${path.module}/k8s-schema.json"
  }

  # TOML validation (auto-detected from .toml extension)
  data "jsonschema_validator" "app_config" {
    document = "${path.module}/config.toml"
    schema   = "${path.module}/config-schema.json"
  }

  # Use the validated document
  locals {
    config = jsondecode(data.jsonschema_validator.config.valid_json)
  }

Standalone CLI
--------------

.. code-block:: bash

  # Validate JSON file
  jsonschema-validator --schema config.schema.json config.json

  # Validate YAML file (auto-detected)
  jsonschema-validator --schema k8s.schema.json deployment.yaml

  # Validate TOML file (auto-detected)
  jsonschema-validator --schema app.schema.json config.toml

  # Force file type override
  jsonschema-validator --schema api.schema.json --force-filetype yaml data.txt

  # With JSON5 support
  jsonschema-validator --schema app.schema.json5 app.json5

  # Use configuration file (zero-config mode)
  jsonschema-validator  # Reads .jsonschema-validator.yaml

Documentation
=============

- **Provider Documentation**: `docs/index.md <docs/index.md>`_
- **Data Source Reference**: `docs/data-sources/jsonschema_validator.md <docs/data-sources/jsonschema_validator.md>`_
- **Examples**: `examples/ <examples/>`_ directory
- **Registry Documentation**: |user-docs|_

Key Features
============

Multi-format Support
--------------------

Validate documents in multiple formats against JSON Schema:

.. code-block:: terraform

  # YAML document validation
  data "jsonschema_validator" "k8s_deployment" {
    document = "${path.module}/deployment.yaml"  # Auto-detected from extension
    schema   = "${path.module}/k8s-schema.json"
  }

  # TOML configuration validation
  data "jsonschema_validator" "app_config" {
    document = "${path.module}/config.toml"  # Auto-detected from extension
    schema   = "${path.module}/config-schema.json"
  }

  # Force file type override
  data "jsonschema_validator" "custom" {
    document       = "${path.module}/data.txt"  # File without standard extension
    schema         = "${path.module}/schema.json"
    force_filetype = "yaml"  # Override auto-detection
  }

JSON5 Support
-------------

.. code-block:: terraform

  # Create a JSON5 document file
  resource "local_file" "json5_config" {
    filename = "${path.module}/config.json5"
    content  = <<-EOT
      {
        // JSON5 comments supported
        "ports": [8080, 8081,], // Trailing commas
        config: { enabled: true } // Unquoted keys
      }
    EOT
  }

  data "jsonschema_validator" "json5_config" {
    document = local_file.json5_config.filename
    schema   = "${path.module}/service.schema.json5"
  }

Reference Overrides
-------------------

Redirect remote ``$ref`` URLs to local files for offline validation:

.. code-block:: terraform

  data "jsonschema_validator" "api_request" {
    document = "${path.module}/api-request.json"
    schema   = "${path.module}/schemas/api-request.schema.json"

    ref_overrides = {
      "https://api.example.com/schemas/user.schema.json" = "${path.module}/schemas/user.schema.json"
    }
  }

**Benefits:**

- **Offline validation** - No internet connection required
- **Air-gapped environments** - Works in restricted networks
- **Deterministic builds** - Same inputs = same results
- **No HTTP complexity** - No proxy settings, authentication, or TLS configuration needed

See `examples/ref_overrides/ <examples/ref_overrides/>`_ for a complete example.

Error Templating
----------------

Customize error output with Go templates:

.. code-block:: terraform

  data "jsonschema_validator" "config" {
    document = "${path.module}/config.yaml"
    schema   = "${path.module}/config.schema.json"
    error_message_template = "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}"
  }

Available template variables: ``{{.FullMessage}}``, ``{{.ErrorCount}}``, ``{{.Errors}}``, ``{{.SchemaFile}}``, ``{{.Document}}``

See the `full documentation <docs/index.md>`_ for advanced templating examples.

CLI Tool (Pre-commit Hook)
===========================

The ``jsonschema-validator`` CLI tool provides the same validation capabilities as the Terraform provider, but as a standalone binary for use in **any project type** (Python, Node.js, Go, etc.).

**Unique Features:**

- ✅ **Multi-format Support** - Validate JSON, JSON5, YAML, and TOML files
- ✅ **Auto-detection** - Format determined from file extension
- ✅ **JSON5 Support** - Only JSON Schema validator with native JSON5 support
- ✅ **Zero-config** - Works without configuration files for simple cases
- ✅ **Project config** - Discovers ``.jsonschema-validator.yaml`` or ``pyproject.toml``
- ✅ **Pre-commit integration** - Native support for pre-commit hooks
- ✅ **Batch validation** - Validate multiple files in one command
- ✅ **CI/CD ready** - Proper exit codes for automation

Configuration Discovery
-----------------------

The CLI automatically discovers configuration from multiple sources (in priority order):

1. **Command-line flags** (highest priority)
2. **Environment variables** (``JSONSCHEMA_VALIDATOR_*``)
3. ``.jsonschema-validator.yaml`` in current directory
4. ``pyproject.toml`` section ``[tool.jsonschema-validator]``

Configuration File Format
-------------------------

**.jsonschema-validator.yaml** (recommended):

.. code-block:: yaml

  # Default schema version (same as Terraform provider)
  schema_version: "draft/2020-12"

  # Multiple schema-document mappings
  schemas:
    - path: "config.schema.json"
      documents:
        - "config.json"
        - "config.*.json"

    - path: "api/schemas/request.schema.json"
      documents:
        - "api/requests/*.json"
      ref_overrides:
        "https://example.com/user.json": "./schemas/user.json"

    - path: "k8s.schema.json"
      documents:
        - "manifests/*.yaml"
      force_filetype: "yaml"  # Override auto-detection

  # Custom error template (same as Terraform provider)
  error_template: |
    {{range .Errors}}
    {{.DocumentPath}}: {{.Message}}
    {{end}}

**pyproject.toml** (for Python projects):

.. code-block:: toml

  [tool.jsonschema-validator]
  schema_version = "draft/2020-12"

  [[tool.jsonschema-validator.schemas]]
  path = "config.schema.json"
  documents = ["config.json"]

  [[tool.jsonschema-validator.schemas]]
  path = "api/request.schema.json"
  documents = ["api/requests/*.json"]

  [[tool.jsonschema-validator.schemas]]
  path = "k8s.schema.json"
  documents = ["manifests/*.yaml"]
  force_filetype = "yaml"  # Override auto-detection

  [tool.jsonschema-validator.schemas.ref_overrides]
  "https://example.com/user.json" = "./schemas/user.json"

CLI Usage Examples
------------------

**Basic validation:**

.. code-block:: bash

  # JSON file
  jsonschema-validator --schema config.schema.json config.json

  # YAML file (auto-detected)
  jsonschema-validator --schema k8s.schema.json deployment.yaml

  # TOML file (auto-detected)
  jsonschema-validator --schema app.schema.json config.toml

  # JSON5 support
  jsonschema-validator --schema app.schema.json5 app.json5

  # Force file type override
  jsonschema-validator --schema api.schema.json --force-filetype yaml data.txt

**With configuration file:**

.. code-block:: bash

  # Uses .jsonschema-validator.yaml automatically
  jsonschema-validator

  # Explicit config file
  jsonschema-validator --config custom-config.yaml

  # Override schema version from config
  jsonschema-validator --schema-version draft/2019-09

**Advanced options (matching Terraform provider):**

.. code-block:: bash

  # Specify schema draft version
  jsonschema-validator --schema-version "draft/2020-12" \
    --schema config.schema.json config.yaml

  # Reference overrides (for offline validation)
  jsonschema-validator \
    --schema api.schema.json \
    --ref-override "https://example.com/user.json=./local/user.json" \
    request.json

  # Force file type (override extension-based detection)
  jsonschema-validator \
    --schema config.schema.json \
    --force-filetype yaml \
    config.txt

  # Custom error template
  jsonschema-validator \
    --schema config.schema.json \
    --error-template '{{range .Errors}}{{.DocumentPath}}: {{.Message}}{{end}}' \
    config.json

  # Validate from stdin
  cat config.yaml | jsonschema-validator --schema config.schema.json -

**Environment variables:**

.. code-block:: bash

  export JSONSCHEMA_VALIDATOR_SCHEMA_VERSION="draft/2020-12"
  export JSONSCHEMA_VALIDATOR_SCHEMA="config.schema.json"
  jsonschema-validator config.json

  # Use custom environment variable prefix
  export MY_APP_SCHEMA_VERSION="draft/2020-12"
  export MY_APP_SCHEMA="config.schema.json"
  jsonschema-validator --env-prefix MY_APP_ config.json

Pre-commit Hook Integration
----------------------------

The CLI tool integrates seamlessly with `pre-commit <https://pre-commit.com/>`_ for automated validation in your development workflow.

**Prerequisites:**

Install the CLI tool first:

.. code-block:: bash

  # Install latest version
  go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest

  # Or install specific version
  go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@v0.5.0

**Add to .pre-commit-config.yaml:**

.. code-block:: yaml

  repos:
    - repo: https://github.com/iilei/terraform-provider-jsonschema
      rev: v0.6.0
      hooks:
        - id: jsonschema-validator
          files: '\.(json|json5|yaml|yml|toml)$'
          args: ['-s', 'schemas/my-schema.json', '--ref-overrides', 'https://example.com/schema.json=./local/schema.json']

**Note:** The hook runs in ``manual`` stage by default. Users define file patterns and all CLI arguments in their configuration. The hook uses ``language: system``, which means the ``jsonschema-validator`` binary must be installed and available in ``$PATH``.

**Examples and Troubleshooting:** See ``examples/pre-commit/`` directory for complete configuration examples, troubleshooting tips, and testing instructions.

**Example workflows:**

Python project with ``pyproject.toml``:

.. code-block:: yaml

  # .pre-commit-config.yaml
  repos:
    - repo: https://github.com/iilei/terraform-provider-jsonschema
      rev: v0.6.0
      hooks:
        - id: jsonschema-validator
          # Automatically reads [tool.jsonschema-validator] from pyproject.toml

Multi-language project with explicit config:

.. code-block:: yaml

  # .pre-commit-config.yaml
  repos:
    - repo: https://github.com/iilei/terraform-provider-jsonschema
      rev: v0.6.0
      hooks:
        - id: jsonschema-validator
          name: Validate API requests
          args: ['--schema', 'api/request.schema.json']
          files: '^api/requests/.*\.(json|yaml)$'

        - id: jsonschema-validator
          name: Validate configuration
          args: ['--schema', 'config.schema.json']
          files: '^config\.(json5|yaml|toml)$'

CI/CD Integration
-----------------

**GitHub Actions:**

.. code-block:: yaml

  name: Validate JSON files
  on: [push, pull_request]

  jobs:
    validate:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4

        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version: '1.23'

        - name: Install jsonschema-validator
          run: go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest

        - name: Validate JSON files
          run: jsonschema-validator  # Uses .jsonschema-validator.yaml

**GitLab CI:**

.. code-block:: yaml

  validate-json:
    image: golang:1.23
    stage: test
    script:
      - go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest
      - jsonschema-validator  # Uses .jsonschema-validator.yaml

Exit Codes
----------

- ``0`` - All validations passed
- ``1`` - Validation errors found (schema violations)
- ``2`` - Usage errors (invalid arguments, missing files, etc.)

Development
===========

Requirements: |go|_ 1.25+

.. code-block:: bash

  # Run tests
  go test ./internal/provider/ -v

  # Run acceptance tests
  TF_ACC=1 go test ./internal/provider/ -v

  # View coverage
  go test ./internal/provider/ -coverprofile=coverage.out
  go tool cover -html=coverage.out


.. |terraform| replace:: Terraform
.. _terraform: https://www.terraform.io/

.. |terraform-install-plugin| replace:: install a terraform plugin
.. _terraform-install-plugin: https://www.terraform.io/docs/plugins/basics.html#installing-a-plugin

.. |user-docs| replace:: User Documentation
.. _user-docs: https://registry.terraform.io/providers/iilei/jsonschema/latest/docs

.. |json-schema| replace:: json-schema
.. _json-schema: https://json-schema.org/

.. |terraform-provider-scaffolding| replace:: terraform-provider-scaffolding
.. _terraform-provider-scaffolding: https://github.com/hashicorp/terraform-provider-scaffolding

.. |terraform-publishing-provider| replace:: Publishing Providers
.. _terraform-publishing-provider: https://www.terraform.io/docs/registry/providers/publishing.html

.. |go| replace:: Go
.. _go: https://golang.org/doc/install
