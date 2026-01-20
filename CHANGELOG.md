# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Free Pollinations.ai fallback provider** - When no API key (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or `GEMINI_API_KEY`) is configured, the tool now automatically falls back to the free [Pollinations.ai](https://pollinations.ai) text generation API instead of exiting with an error.
- URL encoding support for Pollinations.ai prompts with `python3` and `sed` fallback.
- Updated `--help` text to document the free fallback option.

## [1.0.0] - Initial Release

### Added

- Natural language to shell command conversion.
- Support for OpenAI, Anthropic, and Gemini API providers.
- Automatic model fallback within each provider.
- Configuration persistence in `~/.x/config`.
- `--verbose` flag for debug output.
- `--version` flag to display version information.
- `--upgrade` flag for self-updating.
- Interactive command confirmation before execution.

