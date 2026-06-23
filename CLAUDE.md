# CLAUDE.md — DevSecOps K8s Platform

> This file is for Claude CLI to understand the full project context.
> Read this file before doing anything in this repository.

---

## 1. What is this project

A **personal DevSecOps lab** built to enterprise 2026 standards, running entirely local on **Mac M4 (ARM64)** at $0 cost. Goal: learn by building, create a strong DevOps/DevSecOps portfolio for job hunting.

**Repo name:** `k8s-devsecops-platform`
**Owner:** BinhMinh04
**Registry:** `ghcr.io/BinhMinh04/k8s-devsecops-platform`

---

## 2. Overall Architecture

```
Developer (git push)
    │
    ▼
app-repo (this repo)
    │  GitHub Actions CI
    │  Gate 1: Gitleaks       → secret scan (full commit history)
    │  Gate 2: Semgrep        → SAST (source code vulnerability)
    │  Gate 3: Trivy fs       → SCA (dependency CVE scan)
    │  Gate 4: Docker buildx  → build multi-arch (arm64 + amd64)
    │  Gate 5: Trivy image    → CVE scan on built image
    │  Gate 6: Cosign         → keyless image signing (OIDC)
    │  Gate 7: Syft           → generate SBOM
    │  Final step: update image.tag in config-repo
    │
    ▼
config-repo (separate repo — ArgoCD watches this)
    │  helm/myapp/values.yaml → image.tag = $GITHUB_SHA
    │
    ▼
ArgoCD (running inside cluster) auto-syncs
    │
    ▼
Kubernetes cluster (k3d local)
    │
    ├── App pod
    │   └── /health, /metrics endpoints
    │
    ├── Kyverno (admission controller)
    │   ├── Block privileged pods
    │   └── Verify Cosign signature (unsigned → rejected)
    │
    ├── Falco (runtime security)
    │   └── Detect shell-in-container, suspicious syscalls
    │
    └── Observability stack
        ├── Prometheus  → scrape /metrics via ServiceMonitor
        ├── Grafana     → dashboard (4 golden signals)
        ├── Loki        → log aggregation via Promtail
        └── Alertmanager → Telegram/Slack webhook alerts
```

---

## 3. Tech Stack

### Local environment
| Tool | Role | Note |
|------|------|------|
| Colima | Container runtime (replaces Docker Desktop) | Must be running before any docker/k8s command |
| k3d | Run K8s cluster locally (k3s in Docker) | `k3d cluster create lab` |
| kubectl | Cluster control (pure CLI, no TUI) | Primary tool for everything K8s |
| Helm | K8s package manager | Used to install apps and infra |

### Application language
- **Go 1.26** — zero external dependencies, compiles to static binary, runs on distroless

### CI/CD
| Tool | Role |
|------|------|
| GitHub Actions | CI pipeline (will be ported to Jenkins later) |
| Docker buildx | Multi-arch build: `linux/amd64,linux/arm64` |
| ArgoCD | GitOps CD — Git is source of truth, CI never touches cluster directly |

### Security tools
| Tool | Stage | Role |
|------|-------|------|
| Gitleaks | Pre-build | Scan secrets in code and full commit history |
| Semgrep OSS | SAST | Source code vulnerability scanning |
| Trivy (fs) | SCA | CVE scan on dependencies/filesystem |
| Trivy (image) | Post-build | CVE scan on Docker image |
| Snyk | SCA supplement | Needs `SNYK_TOKEN` (free tier limited) |
| Syft | Supply chain | Generate SBOM (Software Bill of Materials) |
| Cosign | Supply chain | Keyless image signing via GitHub OIDC |
| Kyverno | Policy-as-Code | Enforce policies + verify Cosign signatures in cluster |
| Falco | Runtime | Detect threats while containers are running |

### Observability
| Tool | Role |
|------|------|
| Prometheus | Metrics collection (scrape /metrics endpoint) |
| Grafana | Dashboards (4 golden signals: latency/traffic/errors/saturation) |
| Loki + Promtail | Log aggregation and querying |
| Tempo | Distributed tracing (added later) |
| OpenTelemetry | Vendor-neutral telemetry standard |
| Alertmanager | Route alerts → Telegram/Slack |

---

## 4. Repository Structure

### app-repo (this repo)
```
k8s-devsecops-platform/
├── CLAUDE.md                    ← this file
├── README.md
├── .gitignore                   # must include: .env, *.key, kubeconfig
├── .gitleaks.toml               # Gitleaks custom rules (if any)
│
├── app/
│   ├── main.go                  # Go app: /health + /metrics + /
│   ├── go.mod                   # module: devsecops-app, go 1.26
│   ├── Dockerfile               # multi-stage + distroless + USER nonroot
│   └── .dockerignore
│
└── .github/
    └── workflows/
        └── ci.yml               # CI: 5 security gates + build + sign + SBOM
```

### config-repo (separate repo — ArgoCD watches this)
```
k8s-devsecops-config/
├── helm/myapp/
│   ├── Chart.yaml
│   ├── values.yaml              # image.tag auto-updated by CI
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── ingress.yaml
│       └── servicemonitor.yaml  # Prometheus scrape config
├── argocd/
│   ├── application.yaml         # ArgoCD Application definition
│   └── app-of-apps.yaml         # (advanced) deploy full stack via GitOps
└── policies/kyverno/
    ├── disallow-privileged.yaml
    └── verify-image-signature.yaml
```

---

## 5. GitOps Principles — Core Rules

```
CI  → NEVER deploy directly to cluster
CI  → only updates image.tag in config-repo
ArgoCD (inside cluster) → pulls config-repo and syncs to cluster
```

**Why separate CI and CD:**
- CI has no cluster credentials → reduced attack surface
- Every deploy change has a Git audit trail
- ArgoCD self-heals: manual changes in cluster → auto-reverted to Git state
- Correlates with DORA elite team performance metrics

---

## 6. Application Details

**File:** `app/main.go`

Endpoints:
- `GET /` → `Hello DevSecOps`
- `GET /health` → `ok` (used for K8s liveness/readiness probes)
- `GET /metrics` → Prometheus text format (scraped by ServiceMonitor)

**Dockerfile principles:**
- Stage 1 (builder): `golang:1.26` — static binary (`CGO_ENABLED=0`)
- Stage 2 (runtime): `gcr.io/distroless/static-debian12:nonroot` — no shell, no extra packages
- User: `nonroot:nonroot` — never run as root
- Result: near-zero CVEs because distroless eliminates all OS packages

---

## 7. CI Pipeline Gates (`ci.yml`)

```
1. secret-scan   → Gitleaks        (fail = secret found in code/history)
2. sast          → Semgrep         (fail = vulnerability in source logic)
3. sca           → Trivy fs        (fail = CRITICAL/HIGH CVE in dependencies)
         ↓ (only runs if all 3 gates above pass — needs: [...])
4. build         → docker buildx   (linux/amd64 + linux/arm64)
5. image-scan    → Trivy image     (fail = CRITICAL CVE in image)
6. sign          → Cosign keyless  (OIDC via GitHub Actions)
7. sbom          → Syft            → upload sbom.json as workflow artifact
8. update-config → clone config-repo, update image.tag, push → triggers ArgoCD
```

**Key:** `exit-code: '1'` on Trivy = pipeline actually fails on CVE, not just warns.

---

## 8. Kubernetes Cluster

**Create cluster:**
```bash
k3d cluster create lab --servers 1 --agents 2 --port "8080:80@loadbalancer"
```

**Cluster topology:**
- 1 server (control plane): `k3d-lab-server-0`
- 2 agents (workers): `k3d-lab-agent-0`, `k3d-lab-agent-1`
- Load balancer: `k3d-lab-serverlb` — maps `localhost:8080` → cluster port 80
- Runtime: k3s (CNCF certified Kubernetes conformant)
- Default ingress: Traefik

**Frequently used kubectl commands:**
```bash
kubectl get nodes -o wide
kubectl get pods -A
kubectl describe pod <name>                    # debug: shows events
kubectl logs <pod> -f                          # follow logs
kubectl get events --sort-by=.lastTimestamp    # chronological events
kubectl exec -it <pod> -- sh                   # exec into container (Falco detects this)
kubectl rollout status deploy/<name>           # watch rollout progress
kubectl rollout undo deploy/<name>             # rollback
```

**Cluster management:**
```bash
k3d cluster list
k3d cluster stop lab      # pause (data preserved)
k3d cluster start lab     # resume
k3d cluster delete lab    # destroy (must recreate from scratch)
```

---

## 9. Mac M4 Environment Notes

- **Architecture:** ARM64 (Apple Silicon)
- **Container runtime:** Colima — must be running before any docker/k8s command
- **Start Colima:** `colima start --cpu 4 --memory 8 --disk 60` (remembers config after first run)
- **Multi-arch:** images built for both `linux/amd64` and `linux/arm64` — production servers are typically AMD64
- **credsStore issue:** `~/.docker/config.json` must NOT contain `"credsStore": "desktop"` — causes Trivy and docker pull failures. Remove with:
  ```bash
  sed -i '' '/"credsStore"/d' ~/.docker/config.json
  ```

---

## 10. Phase Roadmap

| Phase | Content | Status |
|-------|---------|--------|
| 0 | Environment setup (Colima, k3d, kubectl, Helm) | ✅ Done |
| 1 | Go app + distroless Dockerfile + Trivy scan | ✅ Done |
| 2 | CI GitHub Actions: 5 security gates + build + sign + SBOM | 🔄 In progress |
| 3 | GitOps with ArgoCD (config-repo + auto-sync) | ⬜ Pending |
| 4 | Supply Chain: SBOM + Cosign + Kyverno verify | ⬜ Pending |
| 5 | Observability: Prometheus + Grafana + Loki + OTel | ⬜ Pending |
| 6 | Policy & Runtime: Kyverno + Falco | ⬜ Pending |
| 7 | SRE: DORA metrics + SLO/SLI | ⬜ Pending |
| 8 | Platform: Backstage golden path (bonus) | ⬜ Pending |

---

## 11. Conventions

### Commit messages — Conventional Commits
```
feat:     new feature
fix:      bug fix (including CVE fixes)
ci:       CI pipeline changes
docs:     documentation updates
chore:    housekeeping (dep updates, config)
refactor: code improvement without behavior change
test:     add or update tests
```

### Branching
Working directly on `main` (personal lab, single developer).

### Secrets — never commit to repo
Store in GitHub Actions Secrets:
- `GITHUB_TOKEN` — auto-provided, used for ghcr.io push + Gitleaks
- `SNYK_TOKEN` — if Snyk is enabled
- Cosign: no secret needed (keyless via OIDC)

---

## 12. Common Commands

```bash
# Environment
colima status
colima start --cpu 4 --memory 8 --disk 60
k3d cluster list
kubectl get nodes

# Local build & test
cd app
docker build -t devsecops-app:dev .
docker run -d -p 8080:8080 --name myapp devsecops-app:dev
curl localhost:8080/health
curl localhost:8080/metrics
docker stop myapp && docker rm myapp

# Security scan locally (before pushing)
trivy image devsecops-app:dev --severity HIGH,CRITICAL
trivy fs . --severity HIGH,CRITICAL
gitleaks detect --source . -v

# Git workflow
git status
git add <file>
git commit -m "<type>: <message>"
git push
```

---

## 13. References

- ArgoCD: https://argo-cd.readthedocs.io
- Trivy: https://aquasecurity.github.io/trivy
- Cosign/Sigstore: https://docs.sigstore.dev
- Kyverno: https://kyverno.io/docs
- Falco: https://falco.org/docs
- OpenTelemetry: https://opentelemetry.io/docs
- CNCF Landscape: https://landscape.cncf.io
- Google SRE Book (free): https://sre.google/sre-book/table-of-contents
- DORA metrics: https://dora.dev

---

*Last updated: Phase 2 — CI Pipeline in progress*