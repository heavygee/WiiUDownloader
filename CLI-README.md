# WiiUDownloader CLI

A command-line interface for WiiUDownloader that allows you to download Wii U games, updates, DLC, and demos from Nintendo's servers programmatically.

## Features

- Download Wii U titles by Title ID
- List and search available titles
- Decrypt downloaded contents
- Command-line interface suitable for automation and bots
- Progress reporting with ETA
- Support for all title types (games, updates, DLC, demos)

## Building

### Prerequisites

- Go 1.24+
- Python 3 (for generating title database)

### Build Steps

```bash
# Clone the repository
git clone https://github.com/Xpl0itU/WiiUDownloader.git
cd WiiUDownloader

# Run the build script
chmod +x build-cli.sh
./build-cli.sh
```

This will generate a `wiiu-cli` binary.

## Usage

### List Available Titles

```bash
# List all games
./wiiu-cli -list -category game

# List updates
./wiiu-cli -list -category update

# List DLC
./wiiu-cli -list -category dlc

# List demos
./wiiu-cli -list -category demo

# Filter by region
./wiiu-cli -list -category game -region usa
./wiiu-cli -list -category game -region europe
./wiiu-cli -list -category game -region japan
```

### Search Titles

```bash
# Search for games containing "Mario"
./wiiu-cli -search "Mario" -category game

# Search in all categories
./wiiu-cli -search "Zelda" -category all
```

### Download Titles

```bash
# Download a title (basic)
./wiiu-cli -title 00050000101C9500 -output ./downloads

# Download and decrypt
./wiiu-cli -title 00050000101C9500 -output ./downloads -decrypt

# Download, decrypt, and delete encrypted files
./wiiu-cli -title 00050000101C9500 -output ./downloads -decrypt -delete-encrypted
```

### Command Line Options

| Flag | Description | Example |
|------|-------------|---------|
| `-title` | Title ID to download (hexadecimal) | `00050000101C9500` |
| `-output` | Output directory for downloads | `./downloads` |
| `-decrypt` | Decrypt downloaded contents | (flag) |
| `-delete-encrypted` | Delete encrypted files after decryption | (flag) |
| `-list` | List titles in category | (flag) |
| `-search` | Search term for title names | `"Super Mario"` |
| `-category` | Category: game, update, dlc, demo, all | `game` |
| `-region` | Region: japan, usa, europe, all | `usa` |

## Title ID Format

Title IDs are 16-digit hexadecimal numbers. You can find them by:
- Using the `-list` or `-search` commands
- Looking them up on WiiUBrew wiki
- Using the GUI version first to find the ID

Examples of valid Title IDs:
- `00050000101C9500` - Super Mario 3D World (USA)
- `00050000101C9600` - Super Mario 3D World (EUR)
- `0005000E101C9500` - Super Mario 3D World Update (USA)

## Output Structure

Downloaded titles are saved with the following structure:
```
output_directory/
├── title.tmd          # Title metadata
├── title.tik          # Title ticket
├── title.cert         # Certificate
├── 00000000.app       # Main application content
├── 00000001.app       # Additional content
├── 00000002.h3        # Hash file
└── ...                # More content files
```

After decryption (if enabled):
```
output_directory/
├── code/              # Executable code
├── content/           # Game content
├── meta/              # Metadata
└── ...                # Additional decrypted files
```

## Discord Bot Integration

This CLI version is perfect for Discord bots. Here's a basic integration example:

```javascript
const { exec } = require('child_process');
const path = require('path');

// Function to download a Wii U game
function downloadWiiUGame(titleId, outputDir) {
    return new Promise((resolve, reject) => {
        const command = `./wiiu-cli -title ${titleId} -output "${outputDir}" -decrypt -delete-encrypted`;

        exec(command, { cwd: '/path/to/wiiudownloader' }, (error, stdout, stderr) => {
            if (error) {
                reject(error);
                return;
            }
            resolve(stdout);
        });
    });
}

// Discord bot command
client.on('message', async message => {
    if (message.content.startsWith('!download')) {
        const args = message.content.split(' ');
        if (args.length < 2) {
            message.reply('Usage: !download <title_id>');
            return;
        }

        const titleId = args[1];
        const outputDir = `./downloads/${message.author.id}`;

        try {
            message.reply(`Starting download of title ${titleId}...`);
            const result = await downloadWiiUGame(titleId, outputDir);
            message.reply('Download completed successfully!');
        } catch (error) {
            message.reply(`Download failed: ${error.message}`);
        }
    }
});
```

## Error Handling

The CLI will exit with different codes:
- `0`: Success
- `1`: Invalid arguments or setup error
- `130`: Download cancelled (Ctrl+C)

## Troubleshooting

### Common Issues

1. **"title ID is required"**: You must provide a `-title` flag for downloads
2. **Permission denied**: Make sure the output directory is writable
3. **Network errors**: Check your internet connection; downloads may retry automatically
4. **Title not found**: Verify the Title ID is correct and available

### Logs and Debugging

The CLI provides real-time progress updates. For more verbose output, you can redirect stderr:

```bash
./wiiu-cli -title 00050000101C9500 -output ./downloads 2>&1 | tee download.log
```

## Legal Notice

Please make sure to follow all legal and ethical guidelines when using this program. Downloading and using copyrighted material without proper authorization may violate copyright laws in your country.

## License

This program is distributed under the GPLv3 License.
