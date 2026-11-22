# nidapp-recette-app

Small HTTP application for the platform acceptance runs. One Go file, one dependency (the PostgreSQL driver).

| Route | Answer |
|---|---|
| `/` | 200, name and version (constant `version` in `main.go`) |
| `/version` | 200, version only |
| `/healthz` | 200, health probe |
| `/fail` | 500, always |
| `/db` | 200 with the rows of the witness table, 503 when `DATABASE_URL` is not set |

Settings: `PORT` (default 8080), `DATABASE_URL`, `WITNESS_TABLE` (default `temoin`).

The image runs as a non-root user and listens on port 8080.

```sh
docker build -t nidapp-recette-app .
docker run --rm -p 8080:8080 nidapp-recette-app
curl localhost:8080/version
```

To prove a new build: change `version` in `main.go`, commit, push.
