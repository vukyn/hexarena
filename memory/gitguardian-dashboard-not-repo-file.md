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

⚠️ **Gắn loại trừ vào TỪNG REPO, không workspace-wide.** Câu hỏi quyết định là repo đó có **credential sống** hay không, chứ không phải nó tên gì. `**/*_test.go` gần vô hại ở hexarena — binary game standalone: không DB, không SSO, không `.env` triển khai, không khoá provider; "mật khẩu" duy nhất là room password và chính repo ghi rõ nó không phải cơ chế bảo mật. Cùng luật đó trên **một service có credential sống** (DB, SSO, khoá nhà cung cấp, `.env` deploy) là điểm mù THẬT — test fixture đúng là nơi khoá thật hay bị dán nhầm nhất.

⚠️ **Cố ý KHÔNG liệt kê tên repo ở câu trên, và đó là một phần của bài học.** Bản trước có đích danh bốn repo private là "điểm mù thật". Repo này **công khai** (`vukyn/hexarena`), nên câu đó là chỉ dẫn mục tiêu: nói cho người đọc bất kỳ biết kho nào đáng soi và soi ở đâu. Luật phát biểu theo **tính chất** thì vẫn dùng được y hệt — ai áp thì tự soi repo mình — mà không tặng kèm danh sách. Quy tắc chung: một note nằm trong repo public thì mọi câu trong nó cũng public; đừng viết về repo khác thứ mà repo đó không tự nói ra.

**Luật platform đã viết** (header `isme/.gitleaksignore`, `medioa2/.gitleaksignore`): file ignore *không* phải chỗ dập một literal đang sống — literal thấy được thì đặt tên thành hằng fixture hoặc bỏ đi. Chỉ khi **đã đặt tên rồi mà scanner vẫn bắt** (đúng ca này) thì loại trừ mới là lối còn lại. → [[test-credential-literals-rule]] [[gitleaks-silence-is-not-safety]]

Repo private không có branch protection (cần GitHub Pro) → check đỏ **không chặn merge**. #251 và #253 đều đỏ vì cùng lý do rồi merge.

Liên quan: [[hexarena-host-binary]]
