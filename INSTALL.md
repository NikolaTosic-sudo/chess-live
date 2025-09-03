# Running Chess Live with Docker

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
