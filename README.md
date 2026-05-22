# Translation CLI tool

This is a very simple tool for personal use. If you find it useful go ahead and
use or adapt.

## Setup

Requires some setup/permissions:

- Google Translate API permissions via the default GCP account mechanism, and
- GCP project identifier.

Set the GCP parent via environment variable:

```bash
export TRANSLATE_GCP_PARENT="projects/my-project"
```

Or point to a file containing the value (useful for Docker secrets, agenix, etc.):

```bash
export TRANSLATE_GCP_PARENT_FILE="/path/to/secret"
```

## Features

Translate from language to another; supports all Google Translate languages:

```bash
translate fi-FI en-US keittiö
```

If target language is en-US, retrieves extra information (definitions,
synonyms, antonyms) from the Free Dictionary API.

To only look up an English word:

```bash
translate kitchen
```

Misspelled words show suggestions:

```bash
translate excersice
# not found; try: exercice, excursus, exercise, exorcise, excesses
```

## Develop

Run `nix develop` to get a working environment.
