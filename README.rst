=============================
terraform-provider-jsonschema
=============================

.. image:: https://codecov.io/github/iilei/terraform-provider-jsonschema/branch/master/graph/badge.svg
    :target: https://codecov.io/github/iilei/terraform-provider-jsonschema
    :alt: Coverage Status

A |terraform|_ provider for validating JSON and JSON5 documents using |json-schema|_ specifications.

Features
========

- **JSON5 Support**: Validate JSON and JSON5 documents with JSON5 schemas
- **Schema Versions**: Draft 4, 6, 7, 2019-09, and 2020-12 support  
- **External References**: Resolves ``$ref`` URIs including JSON5 files
- **Reference Overrides**: Redirect remote ``$ref`` URLs to local files for offline validation
- **Enhanced Templating**: Flexible error formatting with Go templates
- **Individual Error Access**: Iterate over multiple validation errors
- **Deterministic Output**: Consistent JSON for stable Terraform state

Installation
============

On |terraform|_ versions 0.13+ use:

.. code-block:: terraform

  terraform {
    required_providers {
      jsonschema = {
        source  = "iilei/jsonschema"
        version = "~> 0.5.0"  // Use the latest version
      }
    }
  }

For |terraform|_ versions 0.12 or lower use instructions: |terraform-install-plugin|_

Usage
=====

See |user-docs|_ for details.

Provider Configuration
======================

.. code-block:: terraform

  provider "jsonschema" {
    # Optional: Custom error template (Go templating)
    error_message_template = "{{.FullMessage}}"
  }

Basic Example
=============

.. code-block:: terraform

  data "jsonschema_validator" "config" {
    document = file("${path.module}/config.json")
    schema   = file("${path.module}/config.schema.json")
  }

  # Use the validated document
  output "validated_config" {
    value = data.jsonschema_validator.config.validated
  }

JSON5 Support Example
====================

.. code-block:: terraform

  # Validate a JSON5 document with JSON5 schema
  data "jsonschema_validator" "json5_config" {
    document = <<-EOT
      {
        // JSON5 comments supported
        "name": "my-service",
        "ports": [8080, 8081,], // Trailing commas allowed
        "features": {
          enabled: true,  // Unquoted keys supported
        }
      }
    EOT
    schema = "${path.module}/service.schema.json5"
  }

Error Templating
================

Customize error output with Go templates:

.. code-block:: terraform

  # Default format
  provider "jsonschema" {
    error_message_template = "{{.FullMessage}}"
  }

  # Individual error iteration
  provider "jsonschema" {
    error_message_template = "{{range .Errors}}{{.Path}}: {{.Message}}{{end}}"
  }

  # Custom format with metadata
  data "jsonschema_validator" "config" {
    document = file("config.json")  
    schema   = file("config.schema.json")
    error_message_template = "Found {{.ErrorCount}} errors:\n{{range .Errors}}- {{.Path}}: {{.Message}}\n{{end}}"
  }

Remote Schema References
========================

Redirect remote schema URLs to local files for offline validation:

.. code-block:: terraform

  data "jsonschema_validator" "api_request" {
    document = file("api-request.json")
    schema   = "${path.module}/schemas/api-request.schema.json"
    
    # Map remote URLs to local files
    ref_overrides = {
      "https://api.example.com/schemas/user.json" = "${path.module}/schemas/user.schema.json"
      "https://api.example.com/schemas/product.json" = "${path.module}/schemas/product.schema.json"
    }
  }

This enables:

- **Offline validation**: No internet connection required
- **Air-gapped environments**: Works in restricted networks
- **Version control**: Keep all schemas in your repository
- **Deterministic builds**: Same inputs always produce same results

Template Variables
==================

Available in ``error_message_template``:

- ``{{.FullMessage}}`` - Complete validation error message
- ``{{.ErrorCount}}`` - Number of validation errors  
- ``{{.Errors}}`` - Array of individual validation errors
- ``{{.Document}}`` - The validated document
- ``{{.Schema}}`` - The schema used for validation

Each error in ``{{.Errors}}`` provides:

- ``{{.Message}}`` - Error description
- ``{{.Path}}`` - JSON path where error occurred
- ``{{.SchemaPath}}`` - Schema path of the failing constraint
- ``{{.Value}}`` - The invalid value

Development
===========

Requirements: |go|_ 1.25+

.. code-block:: bash

  # Run tests
  TF_ACC=1 go test ./internal/provider -v


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
