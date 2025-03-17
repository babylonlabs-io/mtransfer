# Multi Transfer - `mtransferd`

## Overview

`mtransferd` is a tool for sending multiple `bank.MsgMultiSend` transactions efficiently.
It is designed to handle large-scale funds transfers by batching transactions and managing key signing securely.

## Installation

To install `mtransferd`, ensure you have Go installed and then install from source:

```sh
# Clone the repository
git clone https://github.com/babylonlabs-io/mtransfer.git
cd mtransfer

# Install the binary
make make install
```

## Usage

Run `mtransferd` with the available commands:

```sh
mtransferd [command]
```

### Available Commands

- `help` - Display help information about commands.
- `init` - Initialize an `.mtransfer` home directory.
- `keys` - Manage keys used for signing transactions.
- `start` - Start the transfer process.

### Configuration

Before running the `start` command, initialize the home directory:

```sh
mtransferd init
```

Manage keys using:

```sh
mtransferd keys add my_key
```

### Transferring the funds

The `start` command begins the transfer of funds using a provided JSON file, a signer key, and a batch size.

```sh
mtransferd start --file transfer.json --from my_key --batch-size 10000
```

#### Flags:

- `--file` (required) - Path to the JSON file with the recipients of funds.
- `--from` (required) - The key name to sign transactions.
- `--batch-size` (required) - Number of transactions per batch (default: 10000).
- `--validate-only` - Run validation to check total coins to be transferred and recipient count without sending transactions.

Example for validation:

```sh
mtransferd start --file transfer.json --batch-size 10000 --from my_key --validate-only
```
