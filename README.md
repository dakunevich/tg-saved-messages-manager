# Telegram Saved Messages Manager

A simple web application to manage your Telegram "Saved Messages". This tool allows you to view your saved messages history and delete them directly from a web interface.

## Features

- **View Saved Messages**: Lists your saved messages with pagination.
- **Rich Media**:
    - **Photos**: Displays images directly in the feed.
    - **Albums**: Groups multiple medias from the same album into a single card.
    - **Link Previews**: Shows rich previews (title, description, thumbnail) for web links.
- **Smart Deletion**: Select and delete messages. Deleting an album removes all associated messages.
- **Activity Log**: Real-time console logging for server operations.
- **Graceful Shutdown**: Safely stops the client and server on `CTRL+C`.

## Prerequisites

- [Go](https://go.dev/) 1.18 or higher.
- A Telegram account.

## Setup

1. **Get Telegram API Credentials**:
   - Log in to your Telegram account at [my.telegram.org](https://my.telegram.org).
   - Go to "API development tools".
   - Create a new application to get your `AppID` and `AppHash`.

2. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd telegram-manager
   ```

3. **Install dependencies**:
   ```bash
   go mod tidy
   ```

## Running the Application

You need to set the `TG_APP_ID` and `TG_APP_HASH` environment variables before running the application. You can also optionally set `PORT` (default is 8080).

### Linux / macOS
```bash
export TG_APP_ID=your_api_id
export TG_APP_HASH=your_api_hash
go run main.go
```

### Windows (PowerShell)
```powershell
$env:TG_APP_ID="your_api_id"
$env:TG_APP_HASH="your_api_hash"
go run main.go
```

First time you run the app, the terminal will prompt you to enter your phone number and the authentication code sent to your Telegram account.

Once running, open your browser and navigate to:
http://localhost:8080

## Project Structure

- `main.go`: Entry point of the application.
- `internal/`:
  - `server/`: HTTP server logic and API handlers.
  - `tg/`: Telegram client wrapper using `gotd`.
- `static/`: Frontend assets (HTML, JS, CSS).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
