---
name: gitguardian-dashboard-not-repo-file
description: "GitGuardian PR check chỉ tắt được ở DASHBOARD; .gitguardian.yaml là config của ggshield CLI, commit vào repo không làm check xanh"
metadata: 
  node_type: memory
  type: reference
  modified: 2026-09-03T08:46:58.282Z
---

**GitGuardian GitHub App check KHÔNG đọc file nào trong repo.** Muốn tắt một finding phải vào dashboard: https://dashboard.gitguardian.com/settings/secrets/filepath-exclusions (Workspace settings → Secrets detection → Filepath exclusions), glob, **gắn được vào từng repo**. Có ô test thử filepath để xác minh. Loại trừ **áp NGƯỢC lại incident đã có** → không cần ignore tay từng cái.

**Why:** đã đo (2026-09-03, hexarena PR #253). `.gitguardian.yaml` là config của **`ggshield` CLI** — docs nói ggshield tìm `./.gitguardian.yaml` trong thư mục làm việc, `secret.ignored_paths` nằm trong đó. Phụ thuộc **chỉ một chiều: dashboard → ggshield** (trang "Dependencies between the dashboard and ggshield" liệt kê ignored/resolved incident + disabled detector là thứ dashboard quyết và CLI tuân theo). Không có chiều ngược. Nên commit `.gitguardian.yaml` = cấu hình cho CLI, check PR vẫn đỏ.

**How to apply:**

⚠️ **gitleaks và GitGuardian bắt KHÁC nhau — đừng suy từ cái này ra cái kia.** Đo trên hexarena: GitGuardian bắt `const fixturePassword = "the-cat-sat-on-the-mat"` là "Generic Password", còn `gitleaks detect` quét 320 commit ra **no leaks found**. Nên thêm `.gitleaks.toml`/`.gitleaksignore` để chữa một finding của GitGuardian là bịt cái không kêu → đúng thứ header `isme/.gitleaksignore` cấm.

⚠️ **Gắn loại trừ vào TỪNG REPO, không workspace-wide.** `**/*_test.go` gần vô hại ở hexarena (binary game standalone: không DB/SSO/`.env`/credential sống; "mật khẩu" duy nhất là room password ghi rõ không phải bảo mật). Cùng luật đó trên **isme/medioa2/gardener/rainy** là điểm mù THẬT — test fixture là đúng nơi key thật bị dán nhầm.

**Luật platform đã viết** (header `isme/.gitleaksignore`, `medioa2/.gitleaksignore`): file ignore *không* phải chỗ dập một literal đang sống — literal thấy được thì đặt tên thành hằng fixture hoặc bỏ đi. Chỉ khi **đã đặt tên rồi mà scanner vẫn bắt** (đúng ca này) thì loại trừ mới là lối còn lại. → [[test-credential-literals-rule]] [[gitleaks-silence-is-not-safety]]

Repo private không có branch protection (cần GitHub Pro) → check đỏ **không chặn merge**. #251 và #253 đều đỏ vì cùng lý do rồi merge.

Liên quan: [[hexarena-host-binary]]
