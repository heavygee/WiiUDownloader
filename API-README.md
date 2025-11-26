# WiiUDownloader API

A REST API for WiiUDownloader that allows programmatic access to Wii U game downloads. Perfect for Discord bots and automation tools.

## Quick Start

### Using Docker (Recommended)

```bash
# Build the API container
./build-api.sh

# Run with docker-compose (includes nginx proxy)
docker-compose up -d

# Or run directly
docker run -p 8080:8080 -v $(pwd)/downloads:/downloads wiiu-api:latest
```

### API Base URL
- **Direct**: `http://localhost:8080/api/`
- **With nginx proxy**: `http://localhost/api/`

## API Endpoints

### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "time": "2025-11-26T19:30:00Z",
  "version": "1.0.0"
}
```

### List Titles
Get a list of available Wii U titles with optional filtering.

```http
GET /api/titles?category=game&region=usa&search=mario
```

**Query Parameters:**
- `category` (optional): `game`, `update`, `dlc`, `demo`, `all` (default: `game`)
- `region` (optional): `japan`, `usa`, `europe`, `all` (default: `all`)
- `search` (optional): Search term for title names

**Response:**
```json
{
  "count": 2,
  "titles": [
    {
      "id": "00050000101C9500",
      "name": "Super Mario 3D World",
      "region": "USA",
      "type": "Game"
    },
    {
      "id": "00050000101C9600",
      "name": "Super Mario 3D World",
      "region": "Europe",
      "type": "Game"
    }
  ]
}
```

### Get Title Information
Get detailed information about a specific title.

```http
GET /api/titles/{title_id}
```

**Example:**
```http
GET /api/titles/00050000101C9500
```

**Response:**
```json
{
  "id": "00050000101C9500",
  "name": "Super Mario 3D World",
  "region": "USA",
  "type": "Game"
}
```

**Error Responses:**
- `400`: Invalid title ID format
- `404`: Title not found

### Start Download
Start a new download job.

```http
POST /api/download
Content-Type: application/json
```

**Request Body:**
```json
{
  "title_id": "00050000101C9500",
  "decrypt": true,
  "delete_encrypted": false
}
```

**Parameters:**
- `title_id` (required): Title ID in hexadecimal format
- `decrypt` (optional): Whether to decrypt downloaded contents (default: `false`)
- `delete_encrypted` (optional): Delete encrypted files after decryption (default: `false`)

**Response:** `202 Accepted`
```json
{
  "job_id": "00050000101C9500_1643123456",
  "status": "started",
  "title": "Super Mario 3D World"
}
```

**Error Responses:**
- `400`: Invalid JSON or missing title_id
- `404`: Title not found

### Get Download Status
Check the status and progress of a download job.

```http
GET /api/download/{job_id}
```

**Example:**
```http
GET /api/download/00050000101C9500_1643123456
```

**Response:**
```json
{
  "id": "00050000101C9500_1643123456",
  "title_id": "00050000101C9500",
  "title_name": "Super Mario 3D World",
  "status": "downloading",
  "progress": 45.2,
  "download_size": 2147483648,
  "downloaded": 972873472,
  "speed": "2.1 MB/s",
  "eta": "15:30",
  "output_dir": "/downloads/00050000101C9500_1643123456",
  "start_time": "2025-11-26T19:30:00Z",
  "decrypt": true,
  "delete_encrypted": false
}
```

**Status Values:**
- `pending`: Job queued but not started
- `downloading`: Currently downloading
- `completed`: Download finished successfully
- `failed`: Download failed (check `error` field)
- `cancelled`: Download was cancelled

### Cancel Download
Cancel a running download job.

```http
DELETE /api/download/{job_id}
```

**Response:** `200 OK`
```json
{
  "status": "cancelled",
  "job_id": "00050000101C9500_1643123456"
}
```

**Error Responses:**
- `404`: Job not found
- `400`: Cannot cancel completed/failed job

## Discord Bot Integration Examples

### Node.js Discord Bot Example

```javascript
const { Client, GatewayIntentBits } = require('discord.js');
const axios = require('axios');

const client = new Client({
    intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMessages, GatewayIntentBits.MessageContent]
});

const API_BASE = 'http://localhost:8080/api';

// Search for games
client.on('messageCreate', async message => {
    if (message.content.startsWith('!search')) {
        const searchTerm = message.content.slice(8).trim();

        try {
            const response = await axios.get(`${API_BASE}/titles`, {
                params: { search: searchTerm, category: 'game' }
            });

            const titles = response.data.titles.slice(0, 5); // Limit to 5 results

            if (titles.length === 0) {
                message.reply('No games found matching that search.');
                return;
            }

            const results = titles.map(title =>
                `**${title.name}**\nID: \`${title.id}\` | Region: ${title.region}`
            ).join('\n\n');

            message.reply(`Found ${response.data.count} games:\n\n${results}`);
        } catch (error) {
            message.reply('Error searching for games.');
        }
    }
});

// Download a game
client.on('messageCreate', async message => {
    if (message.content.startsWith('!download')) {
        const titleId = message.content.slice(10).trim();

        if (!titleId.match(/^[0-9A-F]{16}$/i)) {
            message.reply('Please provide a valid 16-digit hexadecimal title ID.');
            return;
        }

        try {
            const response = await axios.post(`${API_BASE}/download`, {
                title_id: titleId,
                decrypt: true,
                delete_encrypted: true
            });

            const jobId = response.data.job_id;
            message.reply(`Started download for **${response.data.title}**\nJob ID: \`${jobId}\``);

            // Monitor progress
            const progressInterval = setInterval(async () => {
                try {
                    const statusResponse = await axios.get(`${API_BASE}/download/${jobId}`);
                    const status = statusResponse.data;

                    if (status.status === 'completed') {
                        clearInterval(progressInterval);
                        message.reply(`✅ Download completed: **${status.title_name}**`);
                    } else if (status.status === 'failed') {
                        clearInterval(progressInterval);
                        message.reply(`❌ Download failed: ${status.error}`);
                    } else if (status.status === 'downloading') {
                        // Update progress (optional - might spam Discord)
                        // message.edit(`Downloading: ${status.progress.toFixed(1)}% - ${status.speed} - ETA: ${status.eta}`);
                    }
                } catch (error) {
                    clearInterval(progressInterval);
                    console.error('Error checking download status:', error);
                }
            }, 10000); // Check every 10 seconds

        } catch (error) {
            message.reply('Error starting download. Check the title ID and try again.');
        }
    }
});

// Check download status
client.on('messageCreate', async message => {
    if (message.content.startsWith('!status')) {
        const jobId = message.content.slice(8).trim();

        try {
            const response = await axios.get(`${API_BASE}/download/${jobId}`);

            const status = response.data;
            const progressBar = createProgressBar(status.progress);

            let reply = `**${status.title_name}**\n`;
            reply += `Status: ${status.status}\n`;
            reply += `Progress: ${progressBar} ${status.progress.toFixed(1)}%\n`;

            if (status.speed) {
                reply += `Speed: ${status.speed}\n`;
                reply += `ETA: ${status.eta}\n`;
            }

            message.reply(reply);
        } catch (error) {
            message.reply('Job not found or error checking status.');
        }
    }
});

function createProgressBar(progress) {
    const filled = Math.round(progress / 10);
    const empty = 10 - filled;
    return '█'.repeat(filled) + '░'.repeat(empty);
}

client.login('YOUR_BOT_TOKEN');
```

### Python Discord Bot Example

```python
import discord
from discord.ext import commands
import requests
import asyncio
import re

API_BASE = 'http://localhost:8080/api'

bot = commands.Bot(command_prefix='!')

@bot.command()
async def search(ctx, *, search_term):
    """Search for Wii U games"""
    try:
        response = requests.get(f'{API_BASE}/titles',
                              params={'search': search_term, 'category': 'game'})
        data = response.json()

        if data['count'] == 0:
            await ctx.send("No games found matching that search.")
            return

        titles = data['titles'][:5]  # Limit to 5 results
        results = []

        for title in titles:
            results.append(f"**{title['name']}**\nID: `{title['id']}` | Region: {title['region']}")

        await ctx.send(f"Found {data['count']} games:\n\n" + "\n\n".join(results))

    except Exception as e:
        await ctx.send("Error searching for games.")

@bot.command()
async def download(ctx, title_id):
    """Download a Wii U game by title ID"""
    if not re.match(r'^[0-9A-F]{16}$', title_id, re.IGNORECASE):
        await ctx.send("Please provide a valid 16-digit hexadecimal title ID.")
        return

    try:
        response = requests.post(f'{API_BASE}/download', json={
            'title_id': title_id,
            'decrypt': True,
            'delete_encrypted': True
        })

        if response.status_code == 202:
            data = response.json()
            job_id = data['job_id']
            await ctx.send(f"Started download for **{data['title']}**\nJob ID: `{job_id}`")

            # Monitor progress
            while True:
                await asyncio.sleep(10)  # Check every 10 seconds

                try:
                    status_response = requests.get(f'{API_BASE}/download/{job_id}')
                    status = status_response.json()

                    if status['status'] == 'completed':
                        await ctx.send(f"✅ Download completed: **{status['title_name']}**")
                        break
                    elif status['status'] == 'failed':
                        await ctx.send(f"❌ Download failed: {status.get('error', 'Unknown error')}")
                        break
                    elif status['status'] == 'cancelled':
                        await ctx.send("Download was cancelled.")
                        break

                except Exception as e:
                    await ctx.send("Error checking download status.")
                    break
        else:
            await ctx.send("Error starting download. Check the title ID.")

    except Exception as e:
        await ctx.send("Error starting download.")

@bot.command()
async def status(ctx, job_id):
    """Check download status"""
    try:
        response = requests.get(f'{API_BASE}/download/{job_id}')
        status = response.json()

        progress_bar = create_progress_bar(status['progress'])

        reply = f"**{status['title_name']}**\n"
        reply += f"Status: {status['status']}\n"
        reply += f"Progress: {progress_bar} {status['progress']:.1f}%\n"

        if 'speed' in status and status['speed']:
            reply += f"Speed: {status['speed']}\n"
            reply += f"ETA: {status['eta']}\n"

        await ctx.send(reply)

    except Exception as e:
        await ctx.send("Job not found or error checking status.")

def create_progress_bar(progress):
    filled = round(progress / 10)
    empty = 10 - filled
    return '█' * filled + '░' * empty

bot.run('YOUR_BOT_TOKEN')
```

## Docker Deployment

### Production Setup

1. **Clone the repository:**
```bash
git clone https://github.com/heavygee/WiiUDownloader.git
cd WiiUDownloader
```

2. **Build and run:**
```bash
# Build the API
./build-api.sh

# Run with docker-compose (recommended)
docker-compose up -d

# Or run API only
docker run -d \
  --name wiiu-api \
  -p 8080:8080 \
  -v $(pwd)/downloads:/downloads \
  -v $(pwd)/logs:/logs \
  --restart unless-stopped \
  wiiu-api:latest
```

3. **Check health:**
```bash
curl http://localhost:8080/health
```

### Environment Variables

The API server accepts these command-line flags:
- `-port`: Port to run on (default: `8080`)
- `-downloads`: Directory for downloads (default: `./downloads`)

### Volume Mounts

- `/downloads`: Where downloaded games are stored
- `/logs`: Application logs (optional)

## Security Considerations

### For Production Use:

1. **Enable HTTPS** - Use nginx with SSL certificates
2. **API Authentication** - Add API keys or JWT tokens
3. **Rate Limiting** - Prevent abuse with request limits
4. **CORS Policy** - Restrict origins for web usage
5. **Firewall** - Limit access to necessary ports

### Docker Security:

```yaml
# docker-compose.yml (security additions)
services:
  wiiu-api:
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
    user: appuser
```

## Troubleshooting

### Common Issues:

1. **Port already in use:**
   - Change the port: `docker run -p 8081:8080 ...`

2. **Permission denied:**
   - Ensure download directory is writable: `chmod 755 downloads/`

3. **Database not found:**
   - Run: `python3 grabTitles.py`

4. **Container won't start:**
   - Check logs: `docker logs wiiu-api`

### Logs:

```bash
# View API logs
docker logs wiiu-api

# View with docker-compose
docker-compose logs -f wiiu-api
```

## API Response Codes

- `200`: Success
- `202`: Accepted (async operation started)
- `400`: Bad Request (invalid input)
- `404`: Not Found
- `500`: Internal Server Error

## Rate Limits

- No built-in rate limiting (add nginx limits for production)
- Concurrent downloads limited by system resources
- Large downloads may take significant time

## File Structure

Downloaded games are organized by job ID:
```
/downloads/
└── {job_id}/
    ├── title.tmd
    ├── title.tik
    ├── title.cert
    ├── 00000000.app
    └── ... (other content files)
```

Decrypted games (if enabled):
```
/downloads/
└── {job_id}/
    ├── code/
    ├── content/
    └── meta/
```

---

**Note**: This API provides access to Nintendo's servers. Please ensure you follow all legal and ethical guidelines when using this software.
