# Family Hub App（家庭管家 App）規劃文件

> 本文件與現有 repository（ChatGPT Desktop App）無直接關係，純粹作為新專案 Family Hub App 嘅獨立規劃記錄，方便日後 scaffold（搭建）實際 project。

---

## 1. Feature Tree（功能架構樹）

```
Family Hub App
├── 1. Household Setup（家庭設立）
│   ├── 建立 Household（家庭群組）
│   ├── Invite Member（邀請成員）
│   └── Role（角色）：Admin / Member / Child
│
├── 2. Calendar Module（行事曆模組）
│   ├── 2.1 My Page（個人頁）
│   ├── 2.2 Family Page（家庭頁）
│   ├── 2.3 Event CRUD（新增／編輯／刪除事件）
│   ├── 2.4 Visibility Engine（公開／私密同步邏輯）── 詳見第 2 章
│   └── 2.5 Reminder / Notification（提醒通知）
│
├── 3. Inventory Module（物資管理模組）
│   ├── 3.1 Category（分類管理）
│   ├── 3.2 Stock Tracking（庫存追蹤）
│   ├── 3.3 Expiry Alert（到期提醒）
│   └── 3.4 Barcode Input（掃碼輸入）
│
└── 4. Shared Infra（共用基建）
    ├── Auth（登入認證）
    ├── Realtime Sync Engine（實時同步引擎）
    ├── Push Notification（推送通知）
    └── Data Storage（數據儲存）
```

---

## 2. Calendar 私密／公開同步邏輯（Visibility Engine）

### 2.1 需求還原（用戶原述例子）

Amy 個人頁記錄兩個 Event：

| 時間 | 名稱 | 地點 | Public? |
|---|---|---|---|
| 13:00–15:00 | 上學 | abc 地點 | 公開 |
| 16:00–20:00 | 和 Kate 約會 | def 地點 | 私密 |

家庭頁（其他成員視角）應顯示：

| 時間 | Bubble 顯示 |
|---|---|
| 13:00–15:00 | 自訂顏色 bubble + 完整內容（名稱／地點等） |
| 16:00–20:00 | 固定深灰色 bubble，標題強制顯示「私人活動」，無其他內容 |

### 2.2 Event Data Model（事件數據模型）

```
Event
├── id
├── creator_id           → Member          (建立者＝唯一可編輯／刪除人)
├── title
├── location
├── start_time / end_time
├── color                 (creator 自訂顏色，只用於 public 顯示)
├── is_public: boolean    (由建立時 toggle button 決定，預設無 default，強制選)
└── participants[]        → Member[]        (建立時可 invite/tag 指定成員，
                                              public／private 皆適用)
```

> 註：現時架構只有一個 owner／creator 概念（唔分開 owner_id 同 creator_id），因為需求已明確「只有建立者可修改／刪除」。

### 2.3 Rendering Rule（顯示規則，Pseudocode）

```
function renderOnFamilyPage(event, viewer):
    if viewer == event.creator_id:
        return FullDetail(event)              # 建立者一律睇齊全部

    if event.is_public == true:
        return FullDetail(event)              # public：任何家庭成員睇齊全部內容
                                                #   (title, location, time, color, participants)

    else:  # is_public == false
        if viewer in event.participants:
            return FullDetail(event)           # 被 tag／invite 嘅人：睇齊全部內容
        else:
            return Bubble(
                color = FIXED_DARK_GRAY,        # 唔受 creator 自訂顏色影響
                title = "私人活動",
                location = null,
                content = null,
                participants = null,            # 連「邀請咗邊個」都唔顯示
                time = event.start_time..event.end_time  # 時間段照樣佔用，方便睇 availability
            )
```

### 2.4 設計原則

- **雙重身份共存**：同一份 Event 資料，Creator 個人頁永遠 Full Detail；Family 頁根據 `is_public` 及 viewer 是否為 `participants` 動態 mask（遮蔽）內容。
- **時間段一定顯示**：即使私密，時間段（time slot）都要佔用對應 bubble，方便家人判斷該成員是否 available（例如約時間食飯時可避開）。
- **私密內容零洩漏**：私密 event 對非受邀者只暴露「有事」，唔暴露 what／where／who／甚至「邀請咗邊個」。
- **顏色隔離**：私密 bubble（非受邀者視角）固定用同一種深灰色，唔會用 creator 自訂顏色，避免間接洩漏資訊（例如顏色本身可能暗示活動類型）。
- **Admin 無 override 權**：Admin（父母）對子女嘅私密 event 冇特權，睇到嘅版本同其他非受邀家庭成員一樣（灰色 bubble + 「私人活動」）。隱私優先於監管。
- **建立即決定公開性**：建立 event 時必須經由 toggle button 明確選擇 public／private，冇 category-based default。
- **Invite 獨立於 public/private**：任何成員建立 event（public 或 private）均可自由 invite／tag 指定成員；呢個機制同「是否公開」互相獨立嘅兩個維度。
- **編輯／刪除權限單一化**：無論 public 或 private，只有 creator 本人可以修改或刪除該 event；其他人（包括被 tag 嘅 participants 同 Admin）都冇編輯／刪除權。

### 2.5 Permission Matrix（權限總覽表）

| 操作 | Creator（建立者） | Tagged／Invited Participant（受邀者） | 其他家庭成員（含 Admin） |
|---|---|---|---|
| 建立 event（選 public／private，可 invite 成員） | ✅ | — | ✅（任何人皆可建立） |
| 查看 Full Detail — Public event | ✅ | ✅ | ✅ |
| 查看 Full Detail — Private event | ✅ | ✅ | ❌（只見灰色 bubble +「私人活動」） |
| 查看「邊個被邀請」— Private event | ✅ | ✅（見到其他 participants） | ❌ |
| 修改 event | ✅ | ❌ | ❌（Admin 亦無 override） |
| 刪除 event | ✅ | ❌ | ❌（Admin 亦無 override） |

### 2.6 已確認決策（原 Open Decisions，現已由需求方拍板）

1. **監護權例外** → 已確認：Admin 對子女私密 event **無 override 權限**，隱私優先。
2. **Toggle 粒度** → 已確認：每次建立 event 時用專屬 button 逐個選 public／private，**無 category default**。
3. **Participants 標籤** → 已確認：public／private event 皆可於建立時 invite/tag 指定成員；private event 嘅受邀者可見 Full Detail，非受邀者（包括睇唔到邀請名單）只見灰色 bubble。
4. **編輯權限** → 已確認：**只有 creator** 可修改／刪除，不論 public 或 private，不論是否有 participants。

---

## 3. Data Model（整體 ER 關係樹）

```
Household（家庭）
 └── 1:N → Member（成員）
              ├── 1:N → Event（as creator）
              ├── N:N → Event（as participant，經 join table）
              └── 1:N → InventoryItem（新增 / 修改記錄，選配）

Event（事件）
 ├── N:1 → Member（creator，唯一可編輯／刪除者）
 ├── N:N → Member（participants，建立時 invite／tag，public／private 皆適用）
 ├── color, title, location, start_time, end_time
 └── is_public: boolean（建立時經 toggle button 決定，無 default）

InventoryItem（物資項目）
 ├── N:1 → Household
 ├── category
 ├── quantity, unit
 ├── expiry_date
 └── low_stock_threshold
```

---

## 4. 下一步

- [x] 確認第 2.6 章「已確認決策」（監護權、Toggle 粒度、Participants、編輯權限）
- [ ] 產出 Wireframe（線框圖）：個人頁 / 家庭頁切換 tab、bubble 顯示效果、public/private toggle button UI
- [ ] 選定 Tech Stack（技術棧）並 scaffold 專案（獨立於現有 ChatGPT Desktop repo）
- [ ] 設計實際 Database Schema（含 index、sync 策略、participants join table）
