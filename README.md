<div align="center">
<h1>GDI - Grove Developer Interface</h1>
<img src="https://storage.googleapis.com/grove-brand-assets/Presskit/Logo%20Joined-2.png" alt="Grove logo" width="500"/>

</div>
<br/>

## Table of Contents <!-- omit in toc -->

- [Overview](#overview)
- [Usage](#usage)
  - [gdi](#gdi)
  - [gdi git createpr](#gdi-git-createpr)
  - [gdi config](#gdi-config)
- [Configuration](#configuration)


## Overview

The 🌿 Grove Developer Interface (GDI) 🌿 is a command-line tool designed to streamline
internal developer workflows at Grove. GDI is intended to help developers quickly perform
routine operations and maintain consistency across projects.

## Usage

The Grove Developer Interface (GDI) enables streamlined internal development workflows by providing a unified command-line interface **to** manage configuration settings, execute Git operations, and more. Below are tables of available commands and their flags:

### gdi

| Flag         | Type | Required | Description                          |
| ------------ | ---- | -------- | ------------------------------------ |
| -t, --toggle | bool | ❌        | Toggle verbose mode or other options |
| -h, --help   | bool | ❌        | Show help for gdi                    |

### gdi git createpr

| Flag                     | Type   | Required | Description                                                                         |
| ------------------------ | ------ | -------- | ----------------------------------------------------------------------------------- |
| --pr-title (-t)          | string | ✅        | PR title. Will open a draft PR if the string contains [DRAFT] or [WIP]              |
| --target-branch (-b)     | string | ❌        | Target branch (default "main")                                                      |
| --issue (-i)             | int    | ❌        | Issue number                                                                        |
| --dummy (-d)             | bool   | ❌        | Dummy mode. Will print summary to console and clipboard but not open a PR on GitHub |
| --provider-override (-p) | string | ❌        | LLM provider override. Sets the LLM provider only for this request                  |
| --model-override (-m)    | string | ❌        | LLM model override. Sets the LLM model only for this request                        |

### gdi config

| Flag          | Type   | Required | Description                                                                  |
| ------------- | ------ | -------- | ---------------------------------------------------------------------------- |
| --show (-s)   | bool   | ❌        | Show the configuration                                                       |
| --editor (-e) | string | ❌        | Edit the configuration in the given text editor, for example `nano` or `vim` |

To run the interactive editor, run without any flags, ie. `gdi config`.

## Configuration

The configuration is done through a config YAML file located at `~/.config.gdi.yaml`.

An example configuration file can be found at `config/examples/.config.example.yaml`.

Configuration can be updated using the interactive command `gdi config`.

