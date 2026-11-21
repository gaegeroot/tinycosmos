# Deployment

Tiny Cosmos runs on [Fly.io](https://fly.io) and deploys from GitHub Actions. This page covers where configuration lives and how the production environment is set up. Service-specific details are in each service's doc (for the bot, [telegram-bot.md](telegram-bot.md#deploying-to-flyio)).

## Where configuration lives

| Kind | Example | Where | Set with |
|---|---|---|---|
| Local development | dev bot token | `.env` in the repository root (git-ignored, see `.env.example`) | your editor |
| Production runtime secrets | `TELEGRAM_BOT_TOKEN` | Fly secrets, per Fly app | `fly secrets set` |
| Production runtime settings that aren't secret | `TINY_COSMOS_DB_PATH` | `[env]` in the app's `fly.toml` (committed) | a commit |
| Deploy credentials | `FLY_API_TOKEN` | GitHub environment `production` | `gh secret set --env production` |

The rules:

- **Production secrets never live in git or in GitHub.** The app reads them from its environment, and Fly injects them at runtime. GitHub only holds what it needs to deploy.
- **Dev and production never share credentials.** Use a separate Telegram bot for local development (see [telegram-bot.md](telegram-bot.md#running-locally)).
- **Only `main` can deploy.** The `production` environment only accepts deployments from `main`, so a workflow on any other branch can't read `FLY_API_TOKEN`.

## Services

| Service | Fly app | Config | Workflow |
|---|---|---|---|
| Telegram bot (`cmd/bot`) | `tiny-cosmos-bot` | `fly.toml` | `.github/workflows/deploy-bot.yml` |

When `cmd/api` or `cmd/worker` ship, each becomes its own Fly app with its own config (for example `infrastructure/deployment/api.fly.toml`, deployed with `flyctl deploy --config ...`), its own deploy workflow using `environment: production`, and its own deploy token.

## Setting up the `production` environment (one time)

You need the [GitHub CLI](https://cli.github.com) (`gh auth login`) and [flyctl](https://fly.io/docs/flyctl/install/) (`fly auth login`). Run these from the repository root.

1. **Create the environment and restrict it to `main`:**

   ```bash
   gh api -X PUT repos/{owner}/{repo}/environments/production \
     -F 'deployment_branch_policy[protected_branches]=false' \
     -F 'deployment_branch_policy[custom_branch_policies]=true'
   gh api -X POST repos/{owner}/{repo}/environments/production/deployment-branch-policies \
     -f name=main
   ```

   `gh` fills in `{owner}/{repo}` from the git remote. In the browser, the same is under **Settings → Environments → New environment**: name it `production`, then under **Deployment branches and tags** choose **Selected branches and tags** and add `main`.

2. **Store a Fly deploy token as an environment secret.** `fly tokens create deploy` makes a token that can deploy only the app in `fly.toml`, nothing else in your Fly account:

   ```bash
   fly tokens create deploy -x 999999h | gh secret set FLY_API_TOKEN --env production
   ```

3. **If you stored it as a repository secret earlier, delete that copy** so the token only exists in one place:

   ```bash
   gh secret delete FLY_API_TOKEN
   ```

4. **Check:**

   ```bash
   gh secret list --env production   # should list FLY_API_TOKEN
   ```

Optional: under **Settings → Environments → production → Required reviewers**, add yourself to approve each deploy by hand before it runs.

> **Private repository on a free GitHub plan?** Environment secrets and branch restrictions in private repositories may require a paid plan. If `gh secret set --env` fails, store the token as a repository secret instead (`gh secret set FLY_API_TOKEN`). The workflow works unchanged, but any branch's workflows could then read the token.

## Rotating the deploy token

```bash
fly tokens list                                   # find the old token's ID
fly tokens revoke <id>
fly tokens create deploy -x 999999h | gh secret set FLY_API_TOKEN --env production
```
