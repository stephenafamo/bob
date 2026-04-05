---
sidebar_position: 0
description: Extend Bob's code generation with plugins
---

# Plugins

Bob's entire code generation engine is built on a plugin system. Models, factories, enums, and every other output you get from Bob are all implemented as plugins. This means custom plugins are first-class - they use the exact same interfaces and mechanisms as the built-in ones.

## Built-in Plugins

When using the CLI (e.g. `bobgen-psql`), Bob loads several built-in plugins automatically. They are set up through the `plugins.Setup` function and can be configured in the `plugins` section of the configuration file. See [configuration](../code-generation/configuration#plugins-configuration) for details.

Built-in plugins fall into two categories:

- **Standalone output plugins** register their own output directory and generate code in a separate package (e.g. `models`, `enums`, `factory`).
- **Template extension plugins** don't create their own output. Instead, they extend a standalone output by appending templates to it (e.g. `where`, `loaders`, and `joins` all extend the `models` output).

Browse the [`gen/plugins/`](https://github.com/stephenafamo/bob/tree/main/gen/plugins) package for the full list of built-in plugins and their implementations.

## Custom Plugins

You can write your own plugins to extend the code generation. See [Writing Custom Plugins](./writing-custom-plugins) for a step-by-step guide.
