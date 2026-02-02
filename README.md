# Reisekosten

[![Version](https://img.shields.io/badge/version-1.3.0-blue.svg)](CHANGELOG.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A command-line tool for generating monthly German travel expense reports (Reisekostenabrechnung).

## What It Does

Generates two PDF documents per month:

- **Kilometergelderstattung** - Mileage reimbursement (distance × €0.30 per trip)
- **Verpflegungsmehraufwand** - Meal allowance (€14.00 for 8h+ trips)

The tool automatically:
1. Calculates workdays for the specified month (excluding weekends and German holidays for your configured province)
2. Distributes workdays equally among configured customers (round-robin)
3. Generates formatted PDF documents with proper page breaks
4. Sends the PDFs via email
5. Deletes local PDF files after sending

## Installation

```bash
go build -o reisekosten
```

## Usage

```bash
# Generate for current month
./reisekosten

# Generate for specific month
./reisekosten 2/2026
./reisekosten 12/2025

# Show version
./reisekosten --version
```

## Configuration

Copy `config.example.json` to `config.json` and fill in your details:

```bash
cp config.example.json config.json
```

### Configuration File Structure

```json
{
  "smtp": {
    "host": "smtp.example.com",
    "port": 587,
    "username": "your-smtp-username",
    "password": "your-smtp-password"
  },
  "email": {
    "from": "sender@example.com",
    "to": "recipient@example.com"
  },
  "customers": [
    {
      "id": "1",
      "name": "Client Company GmbH",
      "from": "Origin City, Street (Your Company)",
      "to": "Destination City, Street (Client Company)",
      "reason": "Project work",
      "distance": 50,
      "province": "BW"
    }
  ]
}
```

### Configuration Options

#### SMTP Settings

| Field | Description |
|-------|-------------|
| `host` | SMTP server hostname |
| `port` | SMTP server port (typically 587 for TLS) |
| `username` | SMTP authentication username |
| `password` | SMTP authentication password |

#### Email Settings

| Field | Description |
|-------|-------------|
| `from` | Sender email address |
| `to` | Recipient email address |

#### Customers

Each customer represents a client/destination for business trips:

| Field | Description |
|-------|-------------|
| `id` | Customer identifier (appears in document) |
| `name` | Customer/client name (used as section header) |
| `from` | Origin address with company name |
| `to` | Destination address with client name |
| `reason` | Purpose of the trip |
| `distance` | One-way distance in kilometers (used for mileage calculation) |
| `province` | German state code for holiday calculation (see below) |

#### Province Codes (Bundesland)

Each customer can have a different province for holiday calculations. Use the two-letter abbreviation:

| Code | State (German)           | State (English)              |
|------|--------------------------|------------------------------|
| BW   | Baden-Württemberg        | Baden-Württemberg            |
| BY   | Bayern                   | Bavaria                      |
| BE   | Berlin                   | Berlin                       |
| BB   | Brandenburg              | Brandenburg                  |
| HB   | Bremen                   | Bremen                       |
| HH   | Hamburg                  | Hamburg                      |
| HE   | Hessen                   | Hesse                        |
| MV   | Mecklenburg-Vorpommern   | Mecklenburg-Western Pomerania|
| NI   | Niedersachsen            | Lower Saxony                 |
| NW   | Nordrhein-Westfalen      | North Rhine-Westphalia       |
| RP   | Rheinland-Pfalz          | Rhineland-Palatinate         |
| SL   | Saarland                 | Saarland                     |
| SN   | Sachsen                  | Saxony                       |
| ST   | Sachsen-Anhalt           | Saxony-Anhalt                |
| SH   | Schleswig-Holstein       | Schleswig-Holstein           |
| TH   | Thüringen                | Thuringia                    |

If omitted or invalid, defaults to `BW` (Baden-Württemberg).

### Multiple Customers

When multiple customers are configured, workdays are distributed equally using round-robin assignment:

```json
"customers": [
  {
    "id": "1",
    "name": "Client A GmbH",
    "from": "Neuhausen, Amselweg (Your Company GmbH)",
    "to": "Stuttgart, Hauptstr. (Client A GmbH)",
    "reason": "Projektarbeit",
    "distance": 42,
    "province": "BW"
  },
  {
    "id": "2",
    "name": "Client B GmbH",
    "from": "Neuhausen, Amselweg (Your Company GmbH)",
    "to": "Munich, Bahnhofstr. (Client B GmbH)",
    "reason": "Beratung",
    "distance": 120,
    "province": "BY"
  }
]
```

With 20 workdays and 2 customers, each customer gets 10 days. Mileage is calculated per customer based on their distance.

## Excluded Dates

The following dates are automatically excluded:
- Weekends (Saturday, Sunday)
- German public holidays (based on configured province)
- December 24
- December 27-31

## Output

Generated PDF filenames follow this pattern:
- `MM_YYYY_Reisekosten_Kilometergelderstattung.pdf`
- `MM_YYYY_Reisekosten_Verpflegungsmehraufwand.pdf`

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## License

MIT
