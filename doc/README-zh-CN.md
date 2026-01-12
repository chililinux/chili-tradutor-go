`
# 辣椒翻译器 go 🌶️

chili-translator-go 是一个用 Go 编写的通用机器翻译包装器。它旨在翻译脚本（.sh、.py）、文档文件（Markdown）和数据文件（JSON），同时保持变量、链接和技术语法的完整性。

它的主要优点是智能缓存 v2.1.9，它大大减少了网络调用，并通过本地数据重用加快了重复翻译的速度。

## ✨ 特点

* 多格式：支持.sh、.py、.md、.json、.yaml。
* 语法保护：在翻译过程中自动保护 shell 变量（$VAR、${VAR}）、Markdown 链接和字符串占位符。
* 并行翻译：使用 Goroutines 同时处理多种语言（可通过 -j 调整）。
* 带时间戳的持久缓存：在本地存储翻译并管理数据生命周期，从而实现智能清理。
* 进度界面：实时显示每种语言的进度，具有完美的视觉对齐，无论语言代码大小如何（例如 en 与 zh-CN）。

## 🚀 安装

确保已安装 Go 和系统依赖项（gettext、trans）。
```bash
git克隆https://github.com/chililinux/chili-tradutor-go.git
cd 辣椒翻译器-go/src
go build -o chili-translator-go chili-translator-go-v2.1.9.go
sudo mv chili-translator-go /usr/local/bin/
```

## 🛠️ 用法

### 基础翻译
要将文件翻译为标准语言（EN、ES、IT、DE、FR、RU、ZH、JA、KO）：

辣椒翻译器去-i meu_script.sh


###指定语言和引擎

cheli-treducer-go - 和教程.md


### 缓存清除
删除过去 30 天内未使用的缓存条目：

辣椒翻译器去——清理缓存


## ⚙️ 选项（标志）

|旗帜|长|描述 |
| :--- | :--- | :--- |
| -i | --输入文件|用于翻译的源文件。 |
| -e | --发动机|翻译引擎：Google、Bing、Yandex（默认：Google）。 |
| -s | --来源|源语言（例如：pt、en）（默认值：auto）。 |
| -l | --语言 |用逗号或全部分隔的语言列表。 |
| -j | --工作|同声传译数量（默认值：8）。 |
| -f | --力|通过绕过本地缓存强制转换。 |
| | --清理缓存 |删除过时的缓存项目（> 30 天）。 |
| -q | --安静|静默模式（无视觉进展）。 |
| -v | --详细|运行时显示技术详细信息。 |
| -V | --版本 |显示当前版本。 |

## 📁 输出结构

* Scripts/POT：在./pot/中生成.po文件，在./usr/share/locale/中生成.mo二进制文件。
* Markdown：在 ./doc/ 中生成翻译版本（例如：README-en.md）。
* JSON：在./translated/中生成翻译版本。

## 🛡️ 缓存逻辑 (v2.1.9)

缓存存储在 ~/.cache/chili-tradutor-go/cache.json 中。

* 自动迁移：当检测到以前版本（v2.1.8）的记录时，该工具会自动在旧记录上标记当前时间戳，以避免历史数据丢失。
* 自动更新：每次在缓存中找到项目时，都会更新其“上次使用”时间戳，以防止将来自动清理。
* 安全性：通过 --clean-cache 进行清理只会删除实际不再使用的内容，确保您的翻译知识库健康增长。


开发者：Vilmar Catafesta <vcatafesta@gmail.com>
版权所有 © 2023-2026 ChiliLinux 团队
