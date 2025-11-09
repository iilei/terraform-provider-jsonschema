=============================
terraform-provider-jsonschema
=============================

.. image:: https://codecov.io/github/iilei/terraform-provider-jsonschema/branch/master/graph/badge.svg
    :target: https://codecov.io/github/iilei/terraform-provider-jsonschema
    :alt: Coverage Status

A |terraform|_ provider for validating JSON and JSON5 documents using |json-schema|_ specifications.

.. note::
   **Version 0.x Stability**: This provider is in initial development (0.x.x versions). Per `semantic versioning <https://semver.org/#spec-item-4>`_, breaking changes may occur in minor or patch releases. Pin your provider version and review changelogs before upgrading. Stability is expected at version 1.0.0.

Features
========

- **JSON5 Support**: Validate JSON and JSON5 documents with JSON5 schemas
- **Schema Versions**: Draft 4, 6, 7, 2019-09, and 2020-12 support
- **External References**: Resolves ``$ref`` URIs including JSON5 files
- **Reference Overrides**: Redirect remote ``$ref`` URLs to local files for offline validation
- **Enhanced Error Templating**: Flexible error formatting with Go templates
- **Deterministic Output**: Consistent JSON for stable Terraform state

See the `full documentation <docs/index.md>`_ for detailed usage, advanced features, and examples. **Hands-on examples** demonstrating schema traversal, error templating, and reference overrides are available in the `examples/ <examples/>`_ directory.

Installation
============

On |terraform|_ versions 0.13+ use:

.. code-block:: terraform

  terraform {
    required_providers {
      jsonschema = {
        source  = "iilei/jsonschema"
        version = "~> 0.5.0"  # Pin to specific version
      }
    }
  }

For |terraform|_ versions 0.12 or lower, see |terraform-install-plugin|_.

Quick Start
===========

.. code-block:: terraform

  provider "jsonschema" {
    schema_version = "draft/2020-12"  # Optional
  }

  data "jsonschema_validator" "config" {
    document = file("${path.module}/config.json")
    schema   = "${path.module}/config.schema.json"
  }

  # Use the validated document
  output "validated_config" {
    value = data.jsonschema_validator.config.validated
  }

Documentation
=============

- **Provider Documentation**: `docs/index.md <docs/index.md>`_
- **Data Source Reference**: `docs/data-sources/jsonschema_validator.md <docs/data-sources/jsonschema_validator.md>`_
- **Examples**: `examples/ <examples/>`_ directory
- **Registry Documentation**: |user-docs|_

Key Features
============

JSON5 Support
-------------

.. code-block:: terraform

  data "jsonschema_validator" "json5_config" {
    document = <<-EOT
      {
        // JSON5 comments supported
        "ports": [8080, 8081,], // Trailing commas
        config: { enabled: true } // Unquoted keys
      }
    EOT
    schema = "${path.module}/service.schema.json5"
  }

Reference Overrides
-------------------

Redirect remote ``$ref`` URLs to local files for offline validation:

.. code-block:: terraform

  data "jsonschema_validator" "api_request" {
    document = file("api-request.json")
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
    document = file("config.json")
    schema   = "config.schema.json"
    error_message_template = "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}"
  }

Available template variables: ``{{.FullMessage}}``, ``{{.ErrorCount}}``, ``{{.Errors}}``, ``{{.SchemaFile}}``, ``{{.Document}}``

See the `full documentation <docs/index.md>`_ for advanced templating examples.

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
