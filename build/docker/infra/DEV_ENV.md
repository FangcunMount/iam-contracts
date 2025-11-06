Development infra environment (dev.env)

Purpose
-------
This repository includes a sample environment file `dev.env.sample` for the local development Docker Compose infra. Do NOT commit real secrets or credentials into version control.

How to use
---------
1. Copy the sample to create your local env file:

   cp build/docker/infra/dev.env.sample build/docker/infra/dev.env

2. Edit `build/docker/infra/dev.env` and set credentials (MySQL, Redis, etc.) for your local environment.

3. Start the infra with docker-compose (from repo root):

   make dev-infra-up

4. Run the concurrency tests against local infra:

   make dev-test-all

Security notes
--------------
- `build/docker/infra/dev.env` is ignored by git (added to `.gitignore`).
- Never commit real passwords or secrets. If sensitive values were accidentally committed, consider using tools like `git filter-repo` or `BFG` to remove them from history. This requires care and coordination with your team.

If you want help removing secrets from git history, tell me and I can outline a safe step-by-step plan.