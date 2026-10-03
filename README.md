# Chess Live ♟️

A modern web-based chess platform built with **Go**, **HTMX**, **Templ**, and **PostgreSQL**.  
The app supports both **local and online play**, user authentication, match history, and game reviews.

> Docker setup is available and it's publicly hosted

---

<p align="center">
  <img src="assets/images/gopher-chess.png" width="100%" />
</p>

---

<p align="center">
  <img src="assets/images/main-private.png" width="30%" />
  <img src="assets/images/playing.png" width="30%" />
  <img src="assets/images/match-history.png" width="30%" />
</p>

---

## Why Chess ♟️

Simply put, I love playing chess, and it seemed like a great project to combine
my previous knowledge of frontend (with practicing "pure HTML", tailwind and making requests with HTMX),
my new-found knowledge of backend, and my love for chess.

---

## ✨ Features

- **User Accounts**: Login and signup functionality.
- **Play Chess Locally**: Start a match on the same device.
- **Play Chess Online**: Real-time multiplayer powered by **WebSockets**.
- **Match History**: View a list of your past games.
- **Game Review**: Replay old games move by move.

---

## 🛠️ Tech Stack

- **Backend**: [Go](https://go.dev/)
- **Frontend**: [HTMX](https://htmx.org/) + [Templ](https://templ.guide/)
- **Database**: [PostgreSQL](https://www.postgresql.org/)
- **Real-time Communication**: WebSockets

---

## 🚧 Roadmap

- [x] Add **Docker support** for easy local hosting
- [x] Better error handling
- [x] Better use of Go routines
- [x] Finished adding rules to the game
- [x] Tighting WebSocket implementation
- [x] BIG REFACTOR
- [x] Add unit tests
- [x] EVEN BIGGER REFACTOR
- [x] Polishing the game and adding some small features
- [x] Deploy to a public domain - https://chess.pieetas.com/

---

## 📦 Installation

### <img src="https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png" alt="docker" width="40"/> Running with Docker

### First time

```bash
docker compose up --build
```

### Next runs

```bash
docker compose up
```

### 🔄 Rebuilding After Code Changes

If you **or I** make changes to the code, you’ll need to rebuild the Docker image before running the app again:

```bash
docker compose up --build
```

This ensures Docker picks up the new code and dependencies.

- ✅ **First run** → always use `--build`
- ✅ **After any code changes** (your own or pulled from GitHub) → run with `--build`
- ⚡ **No code changes** and just restarting the app → you can skip rebuilding and run:

```bash
docker compose up
```

If you want to explore the code:

```bash
git clone https://github.com/NikolaTosic-sudo/chess-live
cd chess-live
```

## 🤝 Contributing

### Clone the repo

```bash
git clone https://github.com/NikolaTosic-sudo/chess-live.git
cd chess-live
```

### Submit a pull request

If you'd like to contribute, please fork the repository and open a pull request to the `main` branch.

## 📜 License

This project is licensed under the [MIT License](LICENSE).
