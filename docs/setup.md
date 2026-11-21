# Agent setup

How to run Claude Code safely inside a Docker development container. For normal Flutter development, see [DEVELOPMENT.md](DEVELOPMENT.md).

The container is the security boundary. The agent can only see this project folder and its own login credentials. It runs as a normal (non-root) user and can't reach the rest of your computer.

## Build

Run this from the project root on the host:

```bash
USER_ID=$(id -u) GROUP_ID=$(id -g) docker compose -f infrastructure/docker/compose.yaml build agent
```

`USER_ID`/`GROUP_ID` give the container's `agent` user the same UID/GID as you, so files the agent creates are owned by you on the host, not by root.

## Run

```bash
USER_ID=$(id -u) GROUP_ID=$(id -g) docker compose -f infrastructure/docker/compose.yaml run --rm agent
```

Then start the agent in the container shell with `claude`.

| Setting | Value |
|---|---|
| Mounted path | Host project folder (`$PWD`) → `/workspace` (read-write) |
| Credentials volume | Compose-managed `claude-auth` volume → `/claude-auth` |
| Network mode | `bridge` (Docker's default network: outbound internet only, no published ports) |
| User | `agent`, unprivileged, UID/GID from the host, no `sudo` |
| Not mounted | Home directory, SSH keys, cloud credentials, Docker socket |

The first time you run it, log in with `claude` (`/login`). `docker-entrypoint.sh` copies the credentials into the Compose-managed `claude-auth` volume and restores them on later runs, so you don't have to log in again. To revoke the agent's access, run `docker compose -f infrastructure/docker/compose.yaml down -v` and log out of that session on claude.ai.

## Smoke test

Run this inside the container to check the boundaries and confirm the agent can reach the API and read the project:

```bash
echo "user:      $(id)"
echo "workdir:   $(pwd)"
echo "mounts:";  mount | grep -E ' /workspace | /claude-auth ' | awk '{print "  " $3 " (" $5 ")"}'
echo "network:   $(awk 'NR==3{print "default route via eth0 (Docker bridge)"}' /proc/net/route)"
echo "sudo:      $(command -v sudo || echo 'not installed')"
echo "docker.sock: $(ls /var/run/docker.sock 2>/dev/null || echo 'not mounted')"
echo "write /etc: $(touch /etc/smoke 2>/dev/null && echo ALLOWED || echo denied)"
echo "write /workspace: $(touch /workspace/.smoke && rm /workspace/.smoke && echo allowed)"
echo "claude:    $(claude --version)"

claude -p "Smoke test: in one short paragraph, tell me which user you are running as, your current working directory, and what lib/main.dart contains. Do not modify any files." \
  --allowedTools "Bash(id),Bash(pwd),Read"
```

Output from 2026-11-19:

```text
user:      uid=501(agent) gid=20(dialout) groups=20(dialout)
workdir:   /workspace
mounts:
  /claude-auth (ext4)
  /workspace (fakeowner)
network:   default route via eth0 (Docker bridge)
sudo:      not installed
docker.sock: not mounted
write /etc: denied
write /workspace: allowed
claude:    2.1.328 (Claude Code)

I'm running as user `agent` in `/workspace`. `lib/main.dart` is a small Flutter app called "Check In" with Material 3 and a green seed color (`0xFF3D6B5A`). Its home screen is a `WelcomePage` showing a chat-bubble icon, the "Check In" title, and a line explaining the idea: one question a week for you and your friends, and once everyone answers, the answers are shared with the group. Below that are a "Sign up" button and a "See how it works" button, and neither does anything yet (both have empty `onPressed: () {}` handlers). I didn't change any files.
```

(UID 501 / GID 20 come from the macOS host. On Linux they're usually 1000.)

## Security decisions

These choices are provisional. Revisit them as the project grows, and only widen access when the agent has a real, project-specific need for it.

**What can the agent access?**
Only this project folder (read-write, so it can edit code) and its own Claude login in `/claude-auth`. It can't see the rest of the host: no home directory, no `~/.ssh`, no `~/.aws`, no browser data, no other projects.

**Why a container and a non-root user?**
If the agent runs a bad command (by mistake, or because of a prompt injection in something it reads), the damage stays inside a disposable container. Without root or `sudo` it can't change system files (`/etc` and `/usr` aren't writable), and because the Docker socket isn't mounted it can't start containers or escape to the host through Docker. `--rm` throws the container away after each session.

**Why this network mode?**
The agent needs outbound HTTPS to reach the Anthropic API, and to fetch packages. `bridge` gives it that without `--network host`, which would let it reach services on the host's localhost. No ports are published (`-p`), so nothing in the container can be reached from outside.

**Which credentials does it have, and why?**
Only the Claude Code login, kept in a named volume so it never sits in the project folder (where it could get committed or shared). There are no git push credentials, cloud keys or API tokens for other services. The project has no secrets yet, so there's nothing else to protect.

**What risks are still there?**
- **The agent can modify or delete any file in the project.** The project isn't a git repository yet, so there's no easy way to undo a bad edit. Setting up git (and committing before agent sessions) is the most important next step.
- **Outbound network access is unrestricted.** The agent could send project files to any server. The code isn't sensitive yet, but an allowlist (for example, only `api.anthropic.com` and package registries) is worth adding if secrets are added later.
- **The agent can read its own login token** in `/claude-auth`. That's needed for it to work, but it means the token would leak if the agent were tricked into sending it somewhere.
- **No CPU, memory or process limits.** Add `--memory`, `--cpus` and `--pids-limit` to the run command if a runaway process becomes a problem.

**What's deliberately missing?**
The Flutter SDK isn't in the image, so the agent can edit code but can't build, run or test the app. Add it only when the agent needs to run `flutter analyze` or `flutter test` by itself.
