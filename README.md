# Gator - RSS Feed Aggregator CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/gator)](https://goreportcard.com/report/github.com/yourusername/gator)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A lightweight RSS/Atom feed aggregator that runs in your terminal, stores posts in PostgreSQL, and supports feed following.

## Features

- Subscribe to RSS/Atom feeds
- Automatic periodic fetching
- Clean terminal output
- PostgreSQL storage
- User authentication
- Mark posts as read/unread

## Installation

### Prerequisites
- [Go 1.20+](https://go.dev/dl/)
- [PostgreSQL 15+](https://www.postgresql.org/download/)

### Install CLI
 - bash
go install github.com/yourusername/gator@latest

Configuration
Create a config file at ~/.gator/config.json:

json
{
  "database_url": "postgres://user:password@localhost:5432/gator?sslmode=disable",
  "fetch_interval": "30m",
  "default_user": "your_username"
}
Initialize database:

bash
gator reset

Usage
# Add a new feed
gator addfeed "TechCrunch" https://techcrunch.com/feed/

# Start the aggregator (runs in background)
gator agg 1h

# Browse recent posts
gator browse 10

# Follow/unfollow feeds
gator follow https://example.com/feed.xml
gator unfollow https://example.com/feed.xml

# User management
gator register
gator login

Additional
1) Start the Postgres server in the background
=> sudo service postgresql start

2) Enter the psql shell:
=>  sudo -u postgres psql

3) Connect to the new database:
=> \c gator

4) Run the down migration
=> goose postgres postgres://postgres:postgres@localhost:5432/gator?sslmode=disable down

5) Run the up migration
=> goose postgres postgres://postgres:postgres@localhost:5432/gator?sslmode=disable up


==================== Just connect to DB ====================

1) Enter the psql shell:
=>  sudo -u postgres psql

2) Connect to the new database:
=> \c gator

==================== Generate the Go code ====================

1) Generate the Go code with 
=> sqlc generate




