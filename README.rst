=============================
terraform-provider-jsonschema
=============================

.. contents::

Abstract
========

A |terraform|_ provider for validating json files using |json-schema|_.

Installation
============

On |terraform|_ versions 0.13+ use:

.. code-block:: terraform

  terraform {
    required_providers {
      jsonschema = {
        source = "ileil/jsonschema"
      }
    }
  }

Features
========

JSON Schema Validation
---------------------

Basic Example:

.. code-block:: terraform

    # Basic JSON validation
    data "jsonschema_validator" "config" {
      document = file("${path.module}/config.json")
      schema   = file("${path.module}/schema.json")
    }

    # Using JSON5 for improved schema readability
    data "jsonschema_validator" "api_spec" {
      document = file("${path.module}/api-spec.json")
      schema   = file("${path.module}/schema.json5")  # JSON5 supports comments and trailing commas
    }

    # Example schema.json5
    # {
    #   // JSON Schema with comments
    #   type: "object",
    #   required: ["version", "name"],
    #   properties: {
    #     version: { type: "string" },
    #     name: { type: "string" },
    #   },
    # }

    output "is_valid" {
      value = data.jsonschema_validator.config.validated
    }

Schema Reference Resolution
-------------------------

The provider supports resolving ``$ref`` references in JSON schemas. For security reasons,
references are restricted to allowed file paths configured in the provider:

.. code-block:: terraform


    locals {
      # Look for schemas in parent directories
      schema_root = dirname(find_in_parent_folders("schemas"))
    }

    provider "jsonschema" {
      ref_patterns = [
        "${local.schema_root}/**/*.json",  # Allow all schemas under schema root
        "**/*.json"                        # Allow relative paths from schema location
      ]
    }

    data "jsonschema_validator" "values" {
      document = file("${path.module}/values/document.json")
      schema   = file("${local.schema_root}/main.json")
    }



The ``ref_patterns`` use the glob pattern syntax from `gobwas/glob <https://github.com/gobwas/glob>`_.

Security Note: References are restricted to local files matching the configured patterns
to prevent potential security issues like path traversal or remote code loading.

.. |terraform| replace:: Terraform
.. _terraform: https://www.terraform.io
.. |json-schema| replace:: JSON Schema
.. _json-schema: https://json-schema.org/
.. |user-docs| replace:: User Documentation
.. _user-docs: docs/index.md