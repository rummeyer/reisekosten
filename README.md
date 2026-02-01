# Reisekosten

A command-line tool for generating monthly German travel expense reports (Reisekostenabrechnung).

## What It Does

Generates two PDF documents per month:

- **Kilometergelderstattung** - Mileage reimbursement (distance × €0.30 per trip)
- **Verpflegungsmehraufwand** - Meal allowance (€14.00 for 8h+ trips)

The tool automatically:
1. Calculates workdays for the specified month (excluding weekends and German holidays for Baden-Württemberg)
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
      "distance": 50
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
    "distance": 42
  },
  {
    "id": "2",
    "name": "Client B GmbH",
    "from": "Neuhausen, Amselweg (Your Company GmbH)",
    "to": "Munich, Bahnhofstr. (Client B GmbH)",
    "reason": "Beratung",
    "distance": 120
  }
]
```

With 20 workdays and 2 customers, each customer gets 10 days. Mileage is calculated per customer based on their distance.

## Excluded Dates

The following dates are automatically excluded:
- Weekends (Saturday, Sunday)
- German public holidays (Baden-Württemberg)
- December 24
- December 27-31

## Output

Generated PDF filenames follow this pattern:
- `MM_YYYY_Reisekosten_Kilometergelderstattung.pdf`
- `MM_YYYY_Reisekosten_Verpflegungsmehraufwand.pdf`

## License

MIT
