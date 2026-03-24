# s3fm

A terminal-based file manager for Amazon S3. Browse and navigate your S3 buckets and their contents through an interactive TUI.

## What It Does

s3fm lets you explore your AWS S3 buckets directly from the terminal. You can list all available buckets, navigate into them, and browse their directory structure using keyboard controls — no `aws s3 ls` commands needed.

## Build

Requires Go 1.25+.

```bash
go build -o s3fm .
```

## Usage

```bash
# Start at the bucket list view
./s3fm

# Start directly in a specific bucket
./s3fm --bucket my-bucket

# Specify a region and profile
./s3fm --bucket my-bucket --region us-west-2 --profile production
```

## Flags

| Flag       | Default         | Description                                                                 |
|------------|-----------------|-----------------------------------------------------------------------------|
| `--bucket`  | *(none)*        | The S3 bucket to start in. If omitted, s3fm opens a bucket selection view. |
| `--region`  | `us-east-1`     | The AWS region to use.                                                      |
| `--profile` | `vendor-feed`   | The AWS profile to use for credentials.                                     |

## Keyboard Controls

| Key              | Action                                              |
|------------------|-----------------------------------------------------|
| `up` / `k`       | Move cursor up                                      |
| `down` / `j`     | Move cursor down                                    |
| `G`              | Jump to the top of the list                         |
| `g`              | Jump to the bottom of the list                      |
| `yy`             | Copy the S3 path of the item under the cursor to clipboard (`s3://bucket/prefix/object`) |
| `enter`          | Select bucket or navigate into folder               |
| `esc` / `backspace` | Go back to parent directory or bucket list       |
| `pgup`           | Page up                                             |
| `pgdown`         | Page down                                           |
| `?`              | Show keyboard shortcuts help overlay                |
| `q` / `ctrl+c`   | Quit                                                |

## License

Apache 2.0
