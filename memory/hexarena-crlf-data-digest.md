---
name: hexarena-crlf-data-digest
description: "hexarena PR#276 join Mac→Windows bị data_mismatch vì CRLF đổi data digest; ⚠️ .gitattributes KHÔNG chữa checkout cũ; client giờ in digest của nó"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-04T04:23:38.988Z
---

hexarena PR #276 (`e4ba364`, 2026-09-04). Host trên Mac, join từ Windows → **`data_mismatch` dù CÙNG COMMIT**.

**Nguyên nhân, đo được:** repo không có `.gitattributes`; git trên Windows mặc định `core.autocrlf=true` → checkout đổi LF→CRLF **mọi file text**, gồm 15 file JSON mà `seed.DataDigest()` phủ.

```
LF   (mac)   df3bed25a5c5   ← hexarena-host in ra số này
CRLF (win)   f0ea65c2abb0
```

**How to apply:**

⚠️ **Dữ liệu KHÔNG hề khác — 15/15 file parse ra giá trị y hệt dưới CRLF.** JSON cho phép `\r` giữa token, và `\n` trong chuỗi là hai ký tự `\`+`n` nên autocrlf không đụng. Hai máy sẽ đánh **cùng một trận**. Digest từ chối một khác biệt **không hậu quả**.

⚠️ **KHÔNG sửa ở digest.** Nó băm byte vì băm giá trị đã parse đòi một **dạng chuẩn cho từng quyển sách** — việc lớn hơn nhiều — và nó đang làm ĐÚNG việc: byte khác thì đáp khác. Lỗi là repo **để byte khác nhau giữa hai nền tảng vì lý do chẳng liên quan gì đến dữ liệu**. Sửa ở nguồn: `.gitattributes` = `* text=auto eol=lf`.

⚠️ **THÊM `.gitattributes` KHÔNG CHỮA CHECKOUT ĐANG CÓ.** Đo cả hai chiều: pull luật vào cây đã CRLF → **y nguyên hỏng**; công thức chữa được:
```
git rm --cached -r . && git reset --hard     # ⚠️ lệnh 2 xoá thay đổi chưa commit
```
Một `.gitattributes` để người báo lỗi vẫn hỏng là **nửa fix**.

**Tái hiện được trong repo rác** (`git -c core.autocrlf=true clone`): không luật → `.json` và `.golden` đều ra CRLF; có luật → cả hai LF. Nên **golden không cần luật riêng** (kiểm bằng `git check-attr`, không đoán). `git add --renormalize .` stage **rỗng** vì repo đang 0 file CRLF.

**Client giờ in digest của nó** trên màn join: `máy này — data df3bed25a5c5 · build <...>`. Trước đó **chỉ host in**, nên người bị từ chối không so được, phải đi hỏi.
⚠️ `data`/`build` **để nguyên tiếng Anh cả hai sách** — hai câu từ chối đã bảo *"xem dòng **data** ở hai máy"*, dịch nhãn ở đầu này phá đúng chỉ dẫn đó.
⚠️ **Chỉ digest của MÌNH, không bao giờ của peer**: `wire.Refused` chở đúng `Code`, `Welcome` không chở version, và client bị từ chối chẳng bao giờ nhận Welcome. Gắn digest host vào `Refused` **không phải một field**: 4 chỗ dựng qua 3 helper chỉ nhận `Code`, một chỗ bắn khi **không có room nào** để lấy version, và `Version.Check` cấm nói gì về data với peer lệch protocol → là **một luật, không phải một field**.

⚠️ **Vacuity guard phải là HAI assertion khi giá trị là substring**: `Short()` rỗng thì `strings.Contains(body, "")` **đúng với mọi màn hình**. Nên assert độ dài 12 TRƯỚC, rồi assert bản render với giá trị rỗng **vắng mặt**.
⚠️ **Hardcode BUILD string không đo được** — `buildString()` trả đúng `devel` dưới `go test`, mutation xanh. Nửa digest thì không có lỗ đó (test đưa digest bịa vào, bắt được literal).

Liên quan: [[hexarena-data-digest]] [[hexarena-protocol-wordings]] [[hexarena-host-binary]] [[hexarena-pvp-lobby]]
