# Adit - Secure peer to peer file transfer

A command-line interface (CLI) tool built with Go that enables users to securely send files to one another over a WebRTC connection. This project is inspired by [Magic Wormhole](https://github.com/magic-wormhole/magic-wormhole), a similar tool built in Python.

## Features

- **End-to-End Encryption**: Ensures files are securely transmitted between sender and receiver.
- **Peer-to-Peer Transfer**: Utilizes WebRTC to create a direct connection between devices.
- **Cross-Platform Support**: Works on Windows, macOS, and Linux systems.

## Installation

### Prerequisites
- **cURL (for Linux/macOS)** or **PowerShell (for Windows)**: Used for the installation scripts.

### Installation Instructions

#### Linux/macOS

To install the tool on Linux or macOS, run the following command:

```bash
curl -sSL https://raw.githubusercontent.com/YourGitHubUsername/YourProjectName/main/install/install.sh | sudo bash
```

#### Windows

To install the tool on windows, run the following commands within an elevated powershell prompt:

```powershell
iex "& { $(iwr -useb "https://raw.githubusercontent.com/YourGitHubUsername/YourProjectName/main/install/install.ps1") }"
```

## Usage
Once installed, you can use the tool as follows:

#### Sending a file:
```bash
adit -i /path/to/file
```
This will provide you with a code consisting of 5 standard words that the receiver will need to connect to you and collect the file.

#### Receiving a file:
```bash
adit -c chosen.murmuring.germproof.hardwood.chop
```
The collect code will be the code which was given by the sender. It will only be active for as long as the sender is waiting for the connection and will output the file in your current directory.

## Issues and Bug Reporting
If you encounter any issues or bugs, please report them in the [Github issues](https://github.com/Ryan-Har/adit/issues) section of this repository. Your feedback is appreciated and helps improve the project!

## Contributing
Contributions are welcome! Please feel free to submit a pull request or open an issue.

## License
This project is licensed under the MIT License. See the LICENSE file for details.