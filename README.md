# Chess Live ♟️

A modern web-based chess platform built with **Go**, **HTMX**, **Templ**, and **PostgreSQL**.  
The app supports both **local and online play**, user authentication, match history, and game reviews.  

> ⚠️ The project is still under active development. Currently not easily testable/playable.  
> Docker setup and public hosting will be available soon.  

---

![Go Gopher playing chess](https://raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.svg)

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

- [ ] Add **Docker support** for easy local hosting  
- [ ] Deploy to a public domain  
- [ ] Improve playability & testing environment  
- [ ] Add more robust matchmaking features  
- [ ] Add unit tests  
- [ ] Better use of Go routines
- [ ] Tighting WebSocket implementation

---

## 📦 Installation (Coming Soon)

A Docker image will be provided for local setup.  

For now, if you want to explore the code:  

```bash
git clone https://github.com/NikolaTosic-sudo/chess-live
cd chess-live

