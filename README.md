# Commit Author Refresher

Commit Author Refresher is a tool to update the commit author information in a git repository. It takes a configuration file as input and processes each repository specified in the config. The tool rewrites the commit history, changing the author and committer names and emails to the desired values, and pushes the changes to a target repository.

## Installation

To install the Commit Author Refresher, make sure you have [Go](https://golang.org/dl/) installed on your system. Then, clone the repository and build the binary:

```bash
git clone https://github.com/pkarpovich/commit-author-refresher.git
cd commit-author-refresher
go build -o caf
```


## Usage

Create a JSON configuration file named `caf-config.json` with the following structure:

```json
[
  {
    "originalRepo": "https://github.com/originaluser/repo.git",
    "targetRepo": "https://github.com/targetuser/repo.git",
    "author": {
      "name": "New Author Name",
      "email": "new.author@example.com"
    },
    "excludedAuthors": [
      "exclude1@example.com",
      "exclude2@example.com"
    ]
  }
]
```
- `originalRepo`: The URL of the original repository.
- `targetRepo`: The URL of the target repository where the updated commits will be pushed. 
- `author`: An object containing the new author's name and email. 
- `excludedAuthors`: An array of email addresses that should not be changed.

To process all repositories specified in the configuration file, run the caf binary:

```bash
./caf
```

To process a single repository, use the -p or --project flag followed by the project name:

```bash
./caf -p project-name
```

You can also specify an alternate configuration file with the -f or --file flag:

```bash
./caf -f other-config.json
```

## Testing
To run tests for the Commit Author Refresher, execute the go test command in the terminal:
```bash
go test
```

## Disclaimer
This tool rewrites the commit history, which can be destructive. It is recommended to work on a fresh clone of the repository and double-check the results before pushing the changes. Use this tool at your own risk.

## License
This project is released under the [MIT License](LICENSE).
