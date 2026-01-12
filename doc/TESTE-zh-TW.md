`
# 辣椒-翻譯-go 🌶️

chili-translator-go 是一個用 Go 編寫的通用機器翻譯包裝器。它旨在翻譯腳本（.sh、.py）、文檔文件（Markdown）和數據文件（JSON），同時保持變量、鏈接和技術語法的完整性。

它的主要優點是智能緩存 v2.1.9，它大大減少了網絡調用，並通過本地數據重用加快了重複翻譯的速度。

## ✨ 特點

* 多格式：支持.sh、.py、.md、.json、.yaml。
* 語法保護：在翻譯過程中自動保護 shell 變量（$VAR、${VAR}）、Markdown 鏈接和字符串佔位符。
* 並行翻譯：使用 Goroutines 同時處理多種語言（可通過 -j 調整）。
* 帶時間戳的持久緩存：在本地存儲翻譯並管理數據生命週期，從而實現智能清理。
* 漸進式界面：實時顯示每種語言的進度，並具有完美的視覺對齊，無論語言代碼大小如何（例如 en 與 zh-CN）。

## 🚀 安裝

確保已安裝 Go 和系統依賴項（gettext、trans）。
```bash
git clone https://github.com/chililinux/chili-tradutor-go.git
cd chili-tradutor-go/src
go build -o chili-tradutor-go chili-tradutor-go-v2.1.9.go
sudo mv chili-tradutor-go /usr/local/bin/
```

## 🛠️用法

### 基礎翻譯
要將文件翻譯為標準語言（EN、ES、IT、DE、FR、RU、ZH、JA、KO）：

辣椒翻譯器去-i meu_script.sh


### 指定語言和引擎

cheli-treducer-go - 和教程.md


### 緩存清理
刪除過去 30 天內未使用的緩存條目：

辣椒翻譯器去——清理緩存


## ⚙️ 選項（標誌）

|旗幟|長|描述 |
| :--- | :--- | :--- |
| -i | --輸入文件|用於翻譯的源文件。 |
| -e | --發動機|翻譯引擎：Google、Bing、Yandex（默認：Google）。 |
| -s | --來源|源語言（例如：pt、en）（默認值：auto）。 |
| -l | --語言 |用逗號或全部分隔的語言列表。 |
| -j | --工作|同聲傳譯數量（默認值：8）。 |
| -f | --力|通過繞過本地緩存強制轉換。 |
| | --清理緩存 |刪除過時的緩存項目（> 30 天）。 |
| -q | --安靜|靜默模式（無視覺進展）。 |
| -v | --詳細|運行時顯示技術詳細信息。 |
| -V | --版本 |顯示當前版本。 |

## 📁 輸出結構

* Scripts/POT：在 ./pot/ 中生成 .po 文件，在 ./usr/share/locale/ 中生成 .mo 二進製文件。
* Markdown：在 ./doc/ 中生成翻譯版本（例如：README-en.md）。
* JSON：在 ./translated/ 中生成翻譯版本。

## 🛡️ 緩存邏輯 (v2.1.9)

緩存存儲在 ~/.cache/chili-tradutor-go/cache.json 中。

* 自動遷移：當檢測到以前版本（v2.1.8）的記錄時，該工具會自動在舊記錄上標記當前時間戳，以避免歷史數據丟失。
* 自動更新：每次在緩存中找到項目時，都會更新其“上次使用”時間戳，以防止將來自動清除。
* 安全性：通過 --clean-cache 進行清理只會刪除實際不再使用的內容，確保您的翻譯知識庫健康增長。

---
開發者：Vilmar Catafesta <vcatafesta@gmail.com>
版權所有 © 2023-2026 ChiliLinux 團隊
