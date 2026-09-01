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
├── owner_id            → Member
├── title
├── location
├── start_time / end_time
├── color                (owner 自訂顏色)
├── is_public: boolean   (預設 false)
└── participants[]       (未來擴充：tag 其他成員)
```

### 2.3 Rendering Rule（顯示規則，Pseudocode）

```
function renderOnFamilyPage(event, viewer):
    if viewer == event.owner_id:
        return FullDetail(event)          # 自己一律睇齊全部

    if event.is_public == true:
        return Bubble(
            color = event.color,
            title = event.title,
            location = event.location,
            time = event.start_time..event.end_time
        )
    else:  # 私密
        return Bubble(
            color = FIXED_DARK_GRAY,      # 唔受 owner 自訂顏色影響
            title = "私人活動",
            location = null,
            content = null,
            time = event.start_time..event.end_time   # 時間段照樣佔用，方便睇 availability
        )
```

### 2.4 設計原則

- **雙重身份共存**：同一份 Event 資料，Owner 個人頁永遠 Full Detail；Family 頁根據 `is_public` 動態 mask（遮蔽）內容。
- **時間段一定顯示**：即使私密，時間段（time slot）都要佔用對應 bubble，方便家人判斷該成員是否 available（例如約時間食飯時可避開）。
- **私密內容零洩漏**：私密 event 只暴露「有事」，唔暴露 what／where／who。
- **顏色隔離**：私密 bubble 固定用同一種深灰色，唔會用 owner 自訂顏色，避免間接洩漏資訊（例如顏色本身可能暗示活動類型）。

### 2.5 待決策事項（Open Decisions，需要你確認）

1. **監護權例外**：Admin（例如父母）對未成年子女嘅私密 event，是否有權睇 Full Detail？（隱私 vs 監管取捨）
2. **Toggle 粒度**：`is_public` 係 event 層面獨立設定（如現時例子），定係可以設 default（例如「上學」類別預設公開）？
3. **Participants 標籤**：若 Amy tag Kate 做 participant，係咪強制在 Kate 個人頁出現對應 event？若原 event 係私密，Kate 睇到嘅版本應該係 Full 定 Masked？
4. **編輯權限**：私密 event 嘅顏色／時間變動，是否只 owner 可改？

---

## 3. Data Model（整體 ER 關係樹）

```
Household（家庭）
 └── 1:N → Member（成員）
              ├── 1:N → Event（as owner）
              ├── N:N → Event（as participant，經 join table）
              └── 1:N → InventoryItem（新增 / 修改記錄，選配）

Event（事件）
 ├── N:1 → Member（owner）
 ├── N:N → Member（participants）
 ├── color, title, location, start_time, end_time
 └── is_public: boolean

InventoryItem（物資項目）
 ├── N:1 → Household
 ├── category
 ├── quantity, unit
 ├── expiry_date
 └── low_stock_threshold
```

---

## 4. 下一步

- [ ] 確認第 2.5 章「待決策事項」
- [ ] 產出 Wireframe（線框圖）：個人頁 / 家庭頁切換 tab、bubble 顯示效果
- [ ] 選定 Tech Stack（技術棧）並 scaffold 專案（獨立於現有 ChatGPT Desktop repo）
- [ ] 設計實際 Database Schema（含 index、sync 策略）
