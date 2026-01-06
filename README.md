# SmartRun ⚡

**Energy optimization system for UK households on Octopus Agile tariffs**

SmartRun helps you save money on electricity by automatically finding the cheapest times to run your household appliances based on half-hourly Octopus Agile pricing.

## Features

- 🔌 **Smart Scheduling** - Finds cheapest time slots for appliances
- 🌤️ **Weather-Aware** - Recommends line drying on sunny days
- 📱 **Responsive UI** - Works on desktop and mobile
- 🏠 **Local-First** - All data stored locally (SQLite)
- ⚡ **Real-Time Pricing** - Live Octopus Agile prices for all UK regions
- 🔗 **Coupled Appliances** - Pairs washing machine with dryer based on weather

## Quick Start

### Prerequisites

- Go 1.25 or later
- A UK Octopus Energy Agile tariff subscription
- Your Octopus Energy DNO region code (A-P) - [find your region](https://www.energy-stats.uk/dno-region-codes-explained/)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/awaistahir/smart-run.git
   cd smart-run
   ```

2. **Build the application**
   ```bash
   go build -o smartrund ./cmd/smartrund
   ```

3. **Set up configuration (optional)**
   ```bash
   mkdir -p ~/.smartrun
   cp config.example.yaml ~/.smartrun/config.yaml
   # Edit ~/.smartrun/config.yaml with your region and preferences
   ```

4. **Run the server**
   ```bash
   ./smartrund --port 8080
   ```

5. **Access the web interface**
   - Local: http://localhost:8080
   - Mobile (same WiFi): http://YOUR_IP_ADDRESS:8080

### Quick Start (The Easy Way)

**Option 1: Using the Start Script**
```bash
./scripts/start.sh
```
This script checks if you have Go installed, builds the app, and opens your browser.

**Option 2: Using Make**
```bash
make run
```
Standard `make` commands are also supported.

### Manual Installation

## Configuration

### Setting Your Region

**IMPORTANT:** You must configure your Octopus Energy region code to get accurate pricing.

Find your region code at: https://www.energy-stats.uk/dno-region-codes-explained/

Common region codes:
- **A** - Eastern England
- **B** - East Midlands
- **C** - London
- **D** - Merseyside and Northern Wales
- **E** - West Midlands
- **F** - North Eastern England
- **G** - North Western England
- **H** - Southern England
- **J** - South Eastern England
- **K** - Southern Wales
- **L** - South Western England
- **M** - Yorkshire
- **N** - Southern Scotland
- **P** - Northern Scotland

You can set your region in two ways:

1. **Via Web UI** (Recommended)
   - Open http://localhost:8080
   - Go to Settings
   - Select or enter your region code

2. **Via Configuration File**
   - Edit `~/.smartrun/config.yaml`
   - Set `region: "YOUR_REGION_CODE"`

### Location Settings (Optional)

For weather-aware recommendations (like suggesting line drying instead of tumble dryer):

1. Find your coordinates at: https://www.latlong.net/
2. Update in Settings UI or in `~/.smartrun/config.yaml`:
   ```yaml
   latitude: 51.5074
   longitude: -0.1278
   ```

### Configuring Appliances

1. Open the web interface
2. Click "Add Appliance"
3. Configure:
   - **Name** - e.g., "Washing Machine", "Dishwasher"
   - **Cycle Time** - How long it runs (minutes)
   - **Power Consumption** - Estimated kWh per cycle
   - **Control Type** - Smart plug, manual start, or delayed start
   - **Noise Level** - For quiet hours consideration
   - **Priority** - How urgent it is to run

### Setting Quiet Hours

Prevent noisy appliances from running during sleep hours:

1. Go to Settings
2. Add Quiet Hours periods
3. Format: `22:00` to `07:00`
4. Select which days of the week

## Data Storage

**All data is stored locally on your machine.** No data is sent to external services except:
- Octopus Energy API (public, no authentication) for pricing
- Open-Meteo API (public, no authentication) for weather

Your data location: `~/.smartrun/smartrun.db`

### Privacy & Security

- ✅ No API keys required
- ✅ No personal data leaves your device
- ✅ All APIs used are public and don't require authentication
- ✅ Region code is NOT sensitive (publicly available information)
- ✅ Database is stored locally

## Command Line Usage

### Initialize the database
```bash
./smart-run init
```

### Add an appliance
```bash
./smart-run appliance add --name "Dishwasher" --cycle 120 --kwh 1.5
```

### List appliances
```bash
./smart-run appliance list
```

### Fetch prices
```bash
./smart-run fetch --region C --date today
```

### Generate schedule
```bash
./smart-run plan --region C
```

## Development

### Project Structure

```
smart-run/
├── cmd/
│   ├── smart-run/      # CLI application
│   └── smartrund/      # Web server
├── internal/
│   ├── engine/         # Core scheduling logic
│   ├── prices/         # Octopus API client
│   ├── weather/        # Weather fetching
│   ├── store/          # SQLite database
│   └── uiapi/          # HTTP API server
├── web/                # Frontend (HTML/CSS/JS)
├── scripts/            # Helper scripts
│   └── start.sh        # Quick start script
├── config.example.yaml # Example configuration
└── .gitignore          # Excludes sensitive files
```

### Building from Source

```bash
# Build CLI
go build -o smart-run ./cmd/smart-run

# Build server
go build -o smartrund ./cmd/smartrund

# Run tests
go test ./...
```

## API Endpoints

The web server exposes a REST API at `http://localhost:8080/api/`:

- `GET /api/settings` - Get settings
- `PUT /api/settings` - Update settings
- `GET /api/appliances` - List appliances
- `POST /api/appliances` - Add appliance
- `PUT /api/appliances/{id}` - Update appliance
- `DELETE /api/appliances/{id}` - Delete appliance
- `GET /api/recommendations` - Get recommendations (live)

## How It Works

1. **Fetches Pricing** - Gets half-hourly Octopus Agile prices for today and tomorrow
2. **Fetches Weather** - Gets forecast from Open-Meteo (free, public API)
3. **Analyzes Constraints** - Considers your quiet hours, deadlines, and preferences
4. **Finds Optimal Windows** - Calculates cheapest continuous time slots for each appliance
5. **Weather-Aware Decisions** - Suggests alternatives (e.g., line dry vs. tumble dry)
6. **Generates Recommendations** - Shows top 3 options with cost comparison

## Architecture & Design

SmartRun is built as a **local-first, privacy-respecting** application.

- **Frontend**: Standard web technologies (HTML/CSS/JS) served locally.
- **Backend**: Go (Golang) service handling scheduling, Octopus API integration, and SQLite storage.
- **Optimization**: Uses a sliding‑window algorithm to find the cheapest contiguous time slots that satisfy user constraints (quiet hours, finish-by times).
- **Privacy**: All data (household profile, appliance settings) is stored locally in `~/.smartrun/smartrun.db`. No data is sent to the cloud.

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Uses [Octopus Energy API](https://developer.octopus.energy/docs/api/) for pricing data
- Uses [Open-Meteo API](https://open-meteo.com/) for weather forecasts
- Built with [Claude Code](https://claude.com/claude-code)

## Support

For issues, questions, or feature requests, please open an issue on GitHub.

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
