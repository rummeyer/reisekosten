# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.7.0] - 2026-02-02

### Added
- `--config` command line parameter to specify custom config file path

## [1.6.0] - 2026-02-02

### Added
- Config file search in executable directory as fallback

## [1.5.0] - 2026-02-02

### Added
- Configurable `christmasWeekOff` option to exclude Dec 24, 27-31 (default: true)
- Set to `false` in config.json to only exclude public holidays

## [1.4.0] - 2026-02-02

### Changed
- PDFs now generated in memory instead of writing to disk
- Email attachments sent directly from memory streams
- Removed temporary file creation and cleanup

## [1.3.0] - 2026-02-02

### Added
- Per-customer province setting for German state holidays
- Support for all 16 German states (Bundesländer)

### Changed
- Holiday calculation now uses customer's province instead of global setting

## [1.2.0] - 2026-02-01

### Added
- Structured document ID format (RK-YYYY-MM-XXXX) for better sevDesk recognition
- Customer `name` field for section headers
- sevDesk-friendly labels (Beleg-Nr., Datum, Rechnungsart, Abrechnungszeitraum)
- Professional document layout with ASCII separators
- German decimal format for amounts

### Changed
- Improved PDF formatting for OCR compatibility
- Replaced Unicode box characters with ASCII separators

## [1.1.0] - 2026-02-01

### Added
- Customer `distance` field for per-customer mileage calculation
- Multiple customer support with round-robin day distribution
- External configuration file (config.json)
- Smart PDF page breaks (blocks never split across pages)

### Changed
- Refactored code into separate files (document.go, pdf.go, email.go)
- Moved credentials from hardcoded values to config file

### Removed
- Hardcoded SMTP credentials
- Hardcoded customer information

## [1.0.0] - 2026-02-01

### Added
- Initial release
- Generate Kilometergelderstattung PDF
- Generate Verpflegungsmehraufwand PDF
- German holidays support (Baden-Württemberg)
- Automatic email sending via SMTP
- Command-line month/year argument (M/YYYY format)
