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
- `build-txs` - Build unsigned transactions with `bank.MsgMultiSend`.
- `sign-txs` - Sign transactions provided.
- `broadcast-txs`- Broadcast signed transactions.

### Configuration

Before running the `start` command, initialize the home directory:

```sh
mtransferd init
```

Manage keys using:

```sh
mtransferd keys add my_key
```

### Transferring Funds

The `start` command automates the entire fund transfer process, including **transaction generation, signing, and broadcasting**. It requires a JSON file with recipient details, a signer key, and a batch size.

```sh
mtransferd start --file transfer.json --from my_key --batch-size 10000
```

**NOTE:** For reference about the expected file with the transfer recipients information,
check out the `example_transfer.json` file on the root of this repo.

#### Flags:

- `--file` (**required**) - Path to the JSON file containing recipient details.
- `--from` (**required**) - Name of the key used for signing transactions.
- `--batch-size` (**required**) - Number of transactions per batch (**default: 10000**).
- `--validate-only` - Runs validation to check total coins to be transferred and recipient count **without sending transactions**.
- `--start-index` - Specifies the starting index in the recipient list (**default: 0**).

#### Example: Running Validation Only

Use the `--validate-only` flag to verify the transaction details before execution:

```sh
mtransferd start --file transfer.json --batch-size 10000 --from my_key --validate-only
```

### **Alternative Step-by-Step Approach**

If you prefer more control over the process, you can use the following individual commands:

1. **Build Transactions**

   The `build-txs` command generates **unsigned transactions** using `bank.MsgMultiSend`.
   It requires a transfer file, sender key (or address), and batch size.
   For estimating the gas via transaction simulation, the account number and sequence
   are required (specified with the `--account-number` and `--sequence` flags).
   By default, the unsigned transactions are saved in `unsigned_txs.json`.

   #### Usage:

   ```sh
   mtransferd build-txs --file transfer.json --from my_key_or_addr --batch-size 10000 --sequence 1 --account-number 1
   ```

   #### Flags:

   - `--file` (**required**) - Path to the JSON file containing recipient details.
   - `--from` (**required**) - Sender's key name or address.
   - `--batch-size` (**required**) - Number of recipients per transaction.
   - `--account-number` (**required only in online mode**) - Account number corresponding to the sender's account.
   - `--sequence` (**required only in online mode**) - Transaction sequence number corresponding to the sender's account.
   - `--offline` - Enables offline mode, where **gas estimation is based on empirical results** from test runs with different batch sizes. When using this mode, `--account-number` and `--sequence` flags are not required
   - `--validate-only` - Runs validation to check **total coins to be transferred** and **recipient count** without building transactions.
   - `--gas` - Gas limit to be used in all the generated transactions. If not specified, gas will be estimated via simulation (online mode) or base on empirical results (offline mode).
   - `--gas-adjustment` - Adjustment factor to be multiplied against the estimate returned by the tx simulation; if running in offline mode or the gas limit is set manually this flag is ignored.
   - `--start-index`: Start index of the recipient in the list.
   - `--output-file`: Name of the output file where the transactions are saved (default: `unsigned_txs.json`).
   - `--node`: `<host>:<port>` to CometBFT rpc interface for this chain. Defaults to `localhost:26657`

   This command allows you to **prepare transactions before signing and broadcasting**, ensuring they meet the required validation checks.

   > **NOTE**
   >
   > To get the account number and sequence you can query the `x/auth` module,
   > e.g., using the CLI:
   >
   > ```
   > ❯ babylond q auth account bbn1acy96y3qf23nzr02rwqq9yfathy7thexfwvumh --node https://rpc-faucet.testnet.babylonlabs.io
   > account:
   >  type: cosmos-sdk/BaseAccount
   >  value:
   >    account_number: "50221"
   >    address: bbn1acy96y3qf23nzr02rwqq9yfathy7thexfwvumh
   >    public_key:
   >      type: tendermint/PubKeySecp256k1
   >      value: AjwmrNQEImImpaCCH4cqqYB1TzLg4kucMCFm6S1tbHHW
   >    sequence: "528213"
   > ```
   >
   > In this case, the flags should be `--account-number 50221 --sequence 528213`

2. **Sign Transactions:**

   Sign transactions located in the provided file using a signer key. The signed transactions are saved to a `signed_txs.json` file by default.

   **NOTE:** Only offline signing is supported at the moment. Transactions are generated sequentially, starting with the specified sequence number. Each subsequent transaction will use the next sequence number to ensure validity.

   **Example:**

   ```sh
   mtransferd sign-txs --file unsigned_txs.json --from my_key --chain-id bbn-test-1 --offline --account-number 1 --sequence 2
   ```

   **Flags:**

   - `--file` (**required**): Path to the JSON file containing unsigned transactions.
   - `--from` (**required**): Name of the signer key.
   - `--offline` (**required**): Enables offline signing mode.
   - `--account-number` (**required**): Account number of the signer.
   - `--sequence` (**required**): Starting sequence number for the transactions.
   - `--chain-id`: Chain ID for the transactions.
   - `--output-file`: Name of the output file where the signed transactions are saved (default: `signed_txs.json`).
   - `--start-index`: Start index of the transaction in the list to sign.

3. **Broadcast Transactions:**

   Broadcast the signed transactions located in the provided file.

   **Example:**

   ```sh
   mtransferd broadcast-txs --file signed_txs.json
   ```

   **Flags:**

   - `--file` (**required**): Path to the JSON file containing signed transactions.
   - `--start-index`: Start index of the transaction in the list to broadcast.
   - `--node`: `<host>:<port>` to CometBFT rpc interface for this chain. Defaults to `localhost:26657`
